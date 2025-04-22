package handlers

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/iamlongalong/listenmail/pkg/types"
)

// LogHandler 是一个简单的日志处理器
type LogHandler struct {
	logger *log.Logger
}

// NewLogHandler 创建一个新的日志处理器
func NewLogHandler() *LogHandler {
	return &LogHandler{
		logger: log.New(os.Stdout, "[MailLog] ", log.LstdFlags),
	}
}

// Handle 实现 Handler 接口
func (h *LogHandler) Handle(mail *types.Mail) error {
	h.logger.Printf("From: %v", mail.From)
	h.logger.Printf("To: %v", mail.To)
	h.logger.Printf("Subject: %s", mail.Subject)
	h.logger.Printf("Date: %s", mail.Date)
	if len(mail.Attachments) > 0 {
		h.logger.Printf("Attachments: %d", len(mail.Attachments))
	}
	return nil
}

// Match 实现 Handler 接口
func (h *LogHandler) Match(mail *types.Mail) bool {
	return true
}

// SaveAttachmentHandler 是一个保存附件的处理器
type SaveAttachmentHandler struct {
	directory string
}

// NewSaveAttachmentHandler 创建一个新的附件保存处理器
func NewSaveAttachmentHandler(directory string) (*SaveAttachmentHandler, error) {
	if err := os.MkdirAll(directory, 0755); err != nil {
		return nil, fmt.Errorf("create directory error: %v", err)
	}
	return &SaveAttachmentHandler{
		directory: directory,
	}, nil
}

// Handle 实现 Handler 接口
func (h *SaveAttachmentHandler) Handle(mail *types.Mail) error {
	for _, att := range mail.Attachments {
		filename := filepath.Join(h.directory, att.Filename)
		if err := os.WriteFile(filename, att.Data, 0644); err != nil {
			return fmt.Errorf("save attachment error: %v", err)
		}
	}
	return nil
}

// Match 实现 Handler 接口
func (h *SaveAttachmentHandler) Match(mail *types.Mail) bool {
	return len(mail.Attachments) > 0
}

// ForwardHandler 是一个转发邮件的处理器
type ForwardHandler struct {
	forwardTo []string
	// TODO: 实现邮件转发功能
}

// NewForwardHandler 创建一个新的转发处理器
func NewForwardHandler(forwardTo []string) *ForwardHandler {
	return &ForwardHandler{
		forwardTo: forwardTo,
	}
}

// Handle 实现 Handler 接口
func (h *ForwardHandler) Handle(mail *types.Mail) error {
	// TODO: 实现邮件转发逻辑
	return nil
}

// Match 实现 Handler 接口
func (h *ForwardHandler) Match(mail *types.Mail) bool {
	return true
}

// ChainHandler 是一个处理器链，可以按顺序执行多个处理器
type ChainHandler struct {
	handlers []types.Handler
}

// NewChainHandler 创建一个新的处理器链
func NewChainHandler(handlers ...types.Handler) *ChainHandler {
	return &ChainHandler{
		handlers: handlers,
	}
}

// Handle 实现 Handler 接口
func (h *ChainHandler) Handle(mail *types.Mail) error {
	for _, handler := range h.handlers {
		if handler.Match(mail) {
			if err := handler.Handle(mail); err != nil {
				return err
			}
		}
	}
	return nil
}

// Match 实现 Handler 接口
func (h *ChainHandler) Match(mail *types.Mail) bool {
	for _, handler := range h.handlers {
		if handler.Match(mail) {
			return true
		}
	}
	return false
}

// Add 添加一个新的处理器到链中
func (h *ChainHandler) Add(handler types.Handler) *ChainHandler {
	h.handlers = append(h.handlers, handler)
	return h
}
