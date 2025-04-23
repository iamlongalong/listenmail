package sources

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/iamlongalong/listenmail/pkg/types"
	"github.com/iamlongalong/listenmail/pkg/utils"
	"github.com/mailhog/data"
)

// MailHogSource implements a MailHog API client as a mail source
type MailHogSource struct {
	config     *types.MailHogConfig
	client     *http.Client
	dispatcher types.Dispatcher
	done       chan struct{}

	mu           sync.RWMutex
	lastID       data.MessageID
	processedIDs map[data.MessageID]time.Time // 记录已处理的消息ID和处理时间
}

// APIResponse represents the API response from MailHog
type APIResponse struct {
	Total int            `json:"total"`
	Count int            `json:"count"`
	Start int            `json:"start"`
	Items []data.Message `json:"items"`
}

// NewMailHogSource creates a new MailHog source
func NewMailHogSource(config *types.MailHogConfig, dispatcher types.Dispatcher) (*MailHogSource, error) {
	s := &MailHogSource{
		config:       config,
		dispatcher:   dispatcher,
		client:       &http.Client{Timeout: 10 * time.Second},
		done:         make(chan struct{}),
		processedIDs: make(map[data.MessageID]time.Time),
	}

	if config == nil {
		config = &types.MailHogConfig{
			APIURL:   "http://localhost:8025",
			Interval: 5 * time.Second,
		}
	}

	// 启动清理过期记录的goroutine
	go s.cleanProcessedIDs()

	return s, nil
}

// cleanProcessedIDs 定期清理超过24小时的记录
func (s *MailHogSource) cleanProcessedIDs() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.mu.Lock()
			now := time.Now()
			for id, t := range s.processedIDs {
				if now.Sub(t) > 24*time.Hour {
					delete(s.processedIDs, id)
				}
			}
			s.mu.Unlock()
		}
	}
}

// isProcessed 检查消息是否已经处理过
func (s *MailHogSource) isProcessed(id data.MessageID) bool {
	s.mu.RLock()
	_, exists := s.processedIDs[id]
	s.mu.RUnlock()
	return exists
}

// markProcessed 标记消息为已处理
func (s *MailHogSource) markProcessed(id data.MessageID) {
	s.mu.Lock()
	s.processedIDs[id] = time.Now()
	s.mu.Unlock()
}

// Start implements Source interface
func (s *MailHogSource) Start() error {
	if s.config.APIURL == "" {
		return fmt.Errorf("MailHog API URL is required")
	}

	// Start monitoring for new messages
	log.Println("imap source is running...")
	go s.monitor()

	return nil
}

// Stop implements Source interface
func (s *MailHogSource) Stop() error {
	log.Println("imap source is stopping...")
	close(s.done)
	return nil
}

// Name implements Source interface
func (s *MailHogSource) Name() string {
	return s.config.Name
}

func (s *MailHogSource) monitor() {
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

func (s *MailHogSource) checkNewMessages() error {
	resp, err := s.client.Get(fmt.Sprintf("%s/api/v2/messages", s.config.APIURL))
	if err != nil {
		return fmt.Errorf("API request error: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API error: %s", resp.Status)
	}

	var apiResp APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return fmt.Errorf("JSON decode error: %v", err)
	}

	// Process messages in reverse order (oldest first)
	for i := len(apiResp.Items) - 1; i >= 0; i-- {
		msg := apiResp.Items[i]

		// 检查消息是否已处理
		if s.isProcessed(msg.ID) {
			continue
		}

		// Parse the raw message
		mail, err := utils.ParseMail(strings.NewReader(msg.Raw.Data))
		if err != nil {
			continue // Skip this message if we can't parse it
		}

		// Set message ID from MailHog
		mail.ID = string(msg.ID)
		mail.Source = s.Name()

		if err := s.dispatcher.Dispatch(mail); err != nil {
			return fmt.Errorf("dispatch error: %v", err)
		}

		// 标记消息为已处理
		s.markProcessed(msg.ID)

		// 更新最后处理的消息ID
		if i == 0 {
			s.mu.Lock()
			s.lastID = msg.ID
			s.mu.Unlock()
		}
	}

	return nil
}
