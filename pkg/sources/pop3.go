package sources

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/iamlongalong/listenmail/pkg/types"
	"github.com/iamlongalong/listenmail/pkg/utils"
)

// POP3Source implements a POP3 client as a mail source
type POP3Source struct {
	config     *types.POP3Config
	conn       net.Conn
	reader     *bufio.Reader
	dispatcher types.Dispatcher
	done       chan struct{}

	mu            sync.RWMutex
	processedMsgs map[string]time.Time // 记录已处理的消息ID（使用UIDL）和处理时间
}

// NewPOP3Source creates a new POP3 source
func NewPOP3Source(config *types.POP3Config, dispatcher types.Dispatcher) (*POP3Source, error) {
	s := &POP3Source{
		config:        config,
		dispatcher:    dispatcher,
		done:          make(chan struct{}),
		processedMsgs: make(map[string]time.Time),
	}

	if config == nil {
		config = &types.POP3Config{
			TLS:      true,
			Interval: 30 * time.Second,
		}
	}

	// 启动清理过期记录的goroutine
	go s.cleanProcessedMsgs()

	return s, nil
}

// cleanProcessedMsgs 定期清理超过24小时的记录
func (s *POP3Source) cleanProcessedMsgs() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for id, t := range s.processedMsgs {
				if now.Sub(t) > 24*time.Hour {
					delete(s.processedMsgs, id)
				}
			}
			s.mu.Unlock()
		}
	}
}

// isProcessed 检查消息是否已处理过
func (s *POP3Source) isProcessed(id string) bool {
	s.mu.RLock()
	_, exists := s.processedMsgs[id]
	s.mu.RUnlock()
	return exists
}

// markProcessed 标记消息为已处理
func (s *POP3Source) markProcessed(id string) {
	s.mu.Lock()
	s.processedMsgs[id] = time.Now()
	s.mu.Unlock()
}

// Start implements Source interface
func (s *POP3Source) Start() error {
	if s.config.Server == "" {
		return fmt.Errorf("POP3 server address is required")
	}
	if s.config.Username == "" {
		return fmt.Errorf("POP3 username is required")
	}
	if s.config.Password == "" {
		return fmt.Errorf("POP3 password is required")
	}

	var err error
	if s.config.TLS {
		s.conn, err = tls.Dial("tcp", s.config.Server, &tls.Config{})
	} else {
		s.conn, err = net.Dial("tcp", s.config.Server)
	}
	if err != nil {
		return fmt.Errorf("connect error: %v", err)
	}

	s.reader = bufio.NewReader(s.conn)

	// Read greeting
	_, err = s.readResponse()
	if err != nil {
		return fmt.Errorf("read greeting error: %v", err)
	}

	// Login
	if _, err = s.sendCommand(fmt.Sprintf("USER %s", s.config.Username)); err != nil {
		return fmt.Errorf("user command error: %v", err)
	}

	if _, err = s.sendCommand(fmt.Sprintf("PASS %s", s.config.Password)); err != nil {
		return fmt.Errorf("pass command error: %v", err)
	}

	// Start monitoring for new messages
	log.Println("pop3 source is running...")
	go s.monitor()

	return nil
}

// Stop implements Source interface
func (s *POP3Source) Stop() error {
	log.Println("pop3 source is stopping...")
	close(s.done)
	if s.conn != nil {
		s.sendCommand("QUIT")
		s.conn.Close()
	}
	return nil
}

// Name implements Source interface
func (s *POP3Source) Name() string {
	return s.config.Name
}

func (s *POP3Source) monitor() {
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

func (s *POP3Source) checkNewMessages() error {
	// 获取消息列表
	_, err := s.sendCommand("UIDL")
	if err != nil {
		return err
	}

	// 读取UIDL列表
	var uidlList []struct {
		Number int
		ID     string
	}

	for {
		line, err := s.reader.ReadString('\n')
		if err != nil {
			return err
		}
		if line == ".\r\n" {
			break
		}

		var num int
		var id string
		if _, err := fmt.Sscanf(line, "%d %s", &num, &id); err != nil {
			continue
		}
		uidlList = append(uidlList, struct {
			Number int
			ID     string
		}{num, id})
	}

	// 处理每个未处理的消息
	for _, msg := range uidlList {
		// 检查消息是否已处理
		if s.isProcessed(msg.ID) {
			continue
		}

		// 获取消息内容
		_, err = s.sendCommand(fmt.Sprintf("RETR %d", msg.Number))
		if err != nil {
			continue
		}

		// 读取消息内容
		var messageData strings.Builder
		for {
			line, err := s.reader.ReadString('\n')
			if err != nil {
				return err
			}
			if line == ".\r\n" {
				break
			}
			messageData.WriteString(line)
		}

		// 解析消息
		mail, err := utils.ParseMail(strings.NewReader(messageData.String()))
		if err != nil {
			continue
		}

		// 设置唯一ID
		mail.ID = fmt.Sprintf("pop3-%s", msg.ID)

		if err := s.dispatcher.Dispatch(mail); err != nil {
			return err
		}

		// 标记消息为已处理
		s.markProcessed(msg.ID)
	}

	return nil
}

func (s *POP3Source) sendCommand(cmd string) (string, error) {
	_, err := fmt.Fprintf(s.conn, "%s\r\n", cmd)
	if err != nil {
		return "", err
	}
	return s.readResponse()
}

func (s *POP3Source) readResponse() (string, error) {
	line, err := s.reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(line, "+OK") {
		return "", fmt.Errorf("server error: %s", strings.TrimSpace(line))
	}
	return strings.TrimSpace(line), nil
}
