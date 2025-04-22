package sources

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
	"github.com/iamlongalong/listenmail/pkg/types"
	"github.com/iamlongalong/listenmail/pkg/utils"
)

// IMAPSource implements an IMAP client as a mail source
type IMAPSource struct {
	config     *types.IMAPConfig
	client     *client.Client
	dispatcher types.Dispatcher
	done       chan struct{}

	mu            sync.RWMutex
	uidValidity   uint32               // 当前邮箱的UIDVALIDITY
	processedUIDs map[uint32]time.Time // 记录已处理的消息UID和处理时间
	lastUID       uint32               // 最后处理的消息UID
}

// NewIMAPSource creates a new IMAP source
func NewIMAPSource(config *types.IMAPConfig, dispatcher types.Dispatcher) (*IMAPSource, error) {
	s := &IMAPSource{
		config:        config,
		dispatcher:    dispatcher,
		done:          make(chan struct{}),
		processedUIDs: make(map[uint32]time.Time),
	}

	if config == nil {
		config = &types.IMAPConfig{
			TLS:      true,
			Interval: 30 * time.Second,
		}
	}

	// 启动清理过期记录的goroutine
	go s.cleanProcessedUIDs()

	return s, nil
}

// cleanProcessedUIDs 定期清理超过24小时的记录
func (s *IMAPSource) cleanProcessedUIDs() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for uid, t := range s.processedUIDs {
				if now.Sub(t) > 24*time.Hour {
					delete(s.processedUIDs, uid)
				}
			}
			s.mu.Unlock()
		}
	}
}

// isProcessed 检查消息是否已处理过
func (s *IMAPSource) isProcessed(uid uint32) bool {
	s.mu.RLock()
	_, exists := s.processedUIDs[uid]
	s.mu.RUnlock()
	return exists
}

// markProcessed 标记消息为已处理
func (s *IMAPSource) markProcessed(uid uint32) {
	s.mu.Lock()
	s.processedUIDs[uid] = time.Now()
	s.lastUID = uid
	s.mu.Unlock()
}

// Start implements Source interface
func (s *IMAPSource) Start() error {
	if s.config.Server == "" {
		return fmt.Errorf("IMAP server address is required")
	}
	if s.config.Username == "" {
		return fmt.Errorf("IMAP username is required")
	}
	if s.config.Password == "" {
		return fmt.Errorf("IMAP password is required")
	}

	var err error
	if s.config.TLS {
		s.client, err = client.DialTLS(s.config.Server, nil)
	} else {
		s.client, err = client.Dial(s.config.Server)
	}
	if err != nil {
		return fmt.Errorf("connect error: %v", err)
	}

	// Login
	if err := s.client.Login(s.config.Username, s.config.Password); err != nil {
		return fmt.Errorf("login error: %v", err)
	}

	// Start monitoring for new messages
	log.Println("imap source is running...")
	go s.monitor()

	return nil
}

// Stop implements Source interface
func (s *IMAPSource) Stop() error {
	log.Println("smtp source is stopping...")
	close(s.done)
	if s.client != nil {
		s.client.Logout()
	}
	return nil
}

// Name implements Source interface
func (s *IMAPSource) Name() string {
	return s.config.Name
}

func (s *IMAPSource) monitor() {
	ticker := time.NewTicker(s.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			if err := s.checkNewMessages(); err != nil {
				// log error but continue monitoring
				continue
			}
		}
	}
}

func (s *IMAPSource) checkNewMessages() error {
	// Select INBOX
	mbox, err := s.client.Select("INBOX", false)
	if err != nil {
		return err
	}

	// 检查UIDVALIDITY是否改变
	if s.uidValidity != 0 && s.uidValidity != mbox.UidValidity {
		// UIDVALIDITY改变，清空所有记录
		s.mu.Lock()
		s.uidValidity = mbox.UidValidity
		s.processedUIDs = make(map[uint32]time.Time)
		s.lastUID = 0
		s.mu.Unlock()
	}

	// 首次运行，记录UIDVALIDITY
	if s.uidValidity == 0 {
		s.uidValidity = mbox.UidValidity
	}

	// 获取新消息
	if mbox.Messages == 0 {
		return nil
	}

	// 构建搜索条件
	criteria := &imap.SearchCriteria{
		Uid: new(imap.SeqSet),
	}

	s.mu.RLock()
	lastUID := s.lastUID
	s.mu.RUnlock()

	// 如果有最后处理的UID，只获取更新的消息
	if lastUID > 0 {
		criteria.Uid.AddRange(lastUID+1, 0)
	}

	// 搜索新消息
	uids, err := s.client.UidSearch(criteria)
	if err != nil {
		return err
	}

	if len(uids) == 0 {
		return nil
	}

	// 获取消息内容
	seqSet := new(imap.SeqSet)
	seqSet.AddNum(uids...)

	section := &imap.BodySectionName{}
	items := []imap.FetchItem{imap.FetchUid, section.FetchItem()}

	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)
	go func() {
		done <- s.client.UidFetch(seqSet, items, messages)
	}()

	for msg := range messages {
		// 检查消息是否已处理
		if s.isProcessed(msg.Uid) {
			continue
		}

		r := msg.GetBody(section)
		if r == nil {
			continue
		}

		// Parse the message
		mail, err := utils.ParseMail(r)
		if err != nil {
			continue
		}

		// Set a unique ID for the message
		mail.ID = fmt.Sprintf("imap-%d-%d", s.uidValidity, msg.Uid)

		if err := s.dispatcher.Dispatch(mail); err != nil {
			return err
		}

		// 标记消息为已处理
		s.markProcessed(msg.Uid)
	}

	return <-done
}
