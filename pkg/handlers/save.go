package handlers

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/emersion/go-message/mail"
	_ "github.com/mattn/go-sqlite3"

	"github.com/iamlongalong/listenmail/pkg/types"
)

// SaveHandler 将邮件保存到 SQLite 数据库，附件保存到文件系统
type SaveHandler struct {
	db            *sql.DB
	attachmentDir string
}

// SaveConfig 配置 SaveHandler
type SaveConfig struct {
	// SQLite 数据库文件路径
	DBPath string
	// 附件保存目录
	AttachmentDir string
}

// NewSaveHandler 创建一个新的 SaveHandler
func NewSaveHandler(config SaveConfig) (*SaveHandler, error) {
	// 确保附件目录存在
	if err := os.MkdirAll(config.AttachmentDir, 0755); err != nil {
		return nil, fmt.Errorf("create attachment directory error: %v", err)
	}

	// 打开数据库连接
	db, err := sql.Open("sqlite3", config.DBPath)
	if err != nil {
		return nil, fmt.Errorf("open database error: %v", err)
	}

	// 创建表结构
	if err := initDB(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("init database error: %v", err)
	}

	return &SaveHandler{
		db:            db,
		attachmentDir: config.AttachmentDir,
	}, nil
}

// Close 关闭数据库连接
func (h *SaveHandler) Close() error {
	return h.db.Close()
}

// Handle 实现 Handler 接口
func (h *SaveHandler) Handle(mail *types.Mail) error {
	// 开始事务
	tx, err := h.db.Begin()
	if err != nil {
		return fmt.Errorf("begin transaction error: %v", err)
	}
	defer tx.Rollback()

	// 保存邮件基本信息
	mailID, err := h.saveMail(tx, mail)
	if err != nil {
		return fmt.Errorf("save mail error: %v", err)
	}

	// 保存地址信息
	if err := h.saveAddresses(tx, mailID, "from", mail.From); err != nil {
		return fmt.Errorf("save from addresses error: %v", err)
	}
	if err := h.saveAddresses(tx, mailID, "to", mail.To); err != nil {
		return fmt.Errorf("save to addresses error: %v", err)
	}
	if err := h.saveAddresses(tx, mailID, "cc", mail.Cc); err != nil {
		return fmt.Errorf("save cc addresses error: %v", err)
	}
	if err := h.saveAddresses(tx, mailID, "bcc", mail.Bcc); err != nil {
		return fmt.Errorf("save bcc addresses error: %v", err)
	}

	// 保存邮件头
	if err := h.saveHeaders(tx, mailID, mail.Headers); err != nil {
		return fmt.Errorf("save headers error: %v", err)
	}

	// 保存附件
	if err := h.saveAttachments(tx, mailID, mail.Attachments); err != nil {
		return fmt.Errorf("save attachments error: %v", err)
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction error: %v", err)
	}

	return nil
}

// Match 实现 Handler 接口
func (h *SaveHandler) Match(mail *types.Mail) bool {
	return true // 保存所有邮件
}

func initDB(db *sql.DB) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS mails (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			message_id TEXT,
			subject TEXT,
			date DATETIME,
			text_content TEXT,
			html_content TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS addresses (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			mail_id INTEGER,
			type TEXT,
			address TEXT,
			name TEXT,
			FOREIGN KEY(mail_id) REFERENCES mails(id)
		)`,
		`CREATE TABLE IF NOT EXISTS headers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			mail_id INTEGER,
			name TEXT,
			value TEXT,
			FOREIGN KEY(mail_id) REFERENCES mails(id)
		)`,
		`CREATE TABLE IF NOT EXISTS attachments (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			mail_id INTEGER,
			filename TEXT,
			content_type TEXT,
			size INTEGER,
			path TEXT,
			FOREIGN KEY(mail_id) REFERENCES mails(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_mails_message_id ON mails(message_id)`,
		`CREATE INDEX IF NOT EXISTS idx_mails_date ON mails(date)`,
		`CREATE INDEX IF NOT EXISTS idx_addresses_mail_id ON addresses(mail_id)`,
		`CREATE INDEX IF NOT EXISTS idx_headers_mail_id ON headers(mail_id)`,
		`CREATE INDEX IF NOT EXISTS idx_attachments_mail_id ON attachments(mail_id)`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return err
		}
	}

	return nil
}

func (h *SaveHandler) saveMail(tx *sql.Tx, mail *types.Mail) (int64, error) {
	query := `INSERT INTO mails (message_id, subject, date, text_content, html_content)
			 VALUES (?, ?, ?, ?, ?)`

	result, err := tx.Exec(query, mail.MessageID, mail.Subject, mail.Date, mail.Text, mail.HTML)
	if err != nil {
		return 0, err
	}

	return result.LastInsertId()
}

func (h *SaveHandler) saveAddresses(tx *sql.Tx, mailID int64, addrType string, addresses []*mail.Address) error {
	if len(addresses) == 0 {
		return nil
	}

	query := `INSERT INTO addresses (mail_id, type, address, name) VALUES (?, ?, ?, ?)`
	stmt, err := tx.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, addr := range addresses {
		if _, err := stmt.Exec(mailID, addrType, addr.Address, addr.Name); err != nil {
			return err
		}
	}

	return nil
}

func (h *SaveHandler) saveHeaders(tx *sql.Tx, mailID int64, headers map[string][]string) error {
	if len(headers) == 0 {
		return nil
	}

	query := `INSERT INTO headers (mail_id, name, value) VALUES (?, ?, ?)`
	stmt, err := tx.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for name, values := range headers {
		for _, value := range values {
			if _, err := stmt.Exec(mailID, name, value); err != nil {
				return err
			}
		}
	}

	return nil
}

func (h *SaveHandler) saveAttachments(tx *sql.Tx, mailID int64, attachments []types.Attachment) error {
	if len(attachments) == 0 {
		return nil
	}

	query := `INSERT INTO attachments (mail_id, filename, content_type, size, path) VALUES (?, ?, ?, ?, ?)`
	stmt, err := tx.Prepare(query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	// 创建基于日期的子目录
	dateDir := time.Now().Format("2006/01/02")
	attachmentDir := filepath.Join(h.attachmentDir, dateDir)
	if err := os.MkdirAll(attachmentDir, 0755); err != nil {
		return err
	}

	for _, att := range attachments {
		// 生成唯一的文件名
		filename := fmt.Sprintf("%d_%s", mailID, sanitizeFilename(att.Filename))
		path := filepath.Join(dateDir, filename)
		fullPath := filepath.Join(h.attachmentDir, path)

		// 保存文件
		if err := os.WriteFile(fullPath, att.Data, 0644); err != nil {
			return err
		}

		// 记录到数据库
		if _, err := stmt.Exec(mailID, att.Filename, att.ContentType, len(att.Data), path); err != nil {
			return err
		}
	}

	return nil
}

// sanitizeFilename 清理文件名，移除不安全的字符
func sanitizeFilename(filename string) string {
	// 替换不安全的字符
	safe := strings.Map(func(r rune) rune {
		if r > 127 || strings.ContainsRune(`<>:"/\|?*`, r) {
			return '_'
		}
		return r
	}, filename)

	// 确保文件名不为空
	if safe == "" {
		return "unnamed_file"
	}

	return safe
}
