package handlers

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"github.com/iamlongalong/listenmail/pkg/types"
)

// SaveHandler 将邮件保存到 SQLite 数据库，附件保存到文件系统
type SaveHandler struct {
	db            *gorm.DB
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
	db, err := gorm.Open(sqlite.Open(config.DBPath), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open database error: %v", err)
	}

	// 自动迁移表结构
	if err := db.AutoMigrate(&types.DBMail{}, &types.DBAddress{}, &types.DBAttachment{}); err != nil {
		return nil, fmt.Errorf("auto migrate error: %v", err)
	}

	return &SaveHandler{
		db:            db,
		attachmentDir: config.AttachmentDir,
	}, nil
}

// Close 关闭数据库连接
func (h *SaveHandler) Close() error {
	sqlDB, err := h.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Handle 实现 Handler 接口
func (h *SaveHandler) Handle(mail *types.Mail) error {
	// 转换为数据库模型
	dbMail := types.FromMail(mail)

	// 开始事务
	return h.db.Transaction(func(tx *gorm.DB) error {
		// 保存邮件及其关联数据
		if err := tx.Create(dbMail).Error; err != nil {
			return fmt.Errorf("save mail error: %v", err)
		}

		// 保存附件文件
		if err := h.saveAttachmentFiles(dbMail.ID, mail.Attachments); err != nil {
			return fmt.Errorf("save attachment files error: %v", err)
		}

		return nil
	})
}

// Match 实现 Handler 接口
func (h *SaveHandler) Match(mail *types.Mail) bool {
	return true // 保存所有邮件
}

// saveAttachmentFiles 保存附件文件到文件系统
func (h *SaveHandler) saveAttachmentFiles(mailID uint, attachments []types.Attachment) error {
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
	}

	return nil
}

// sanitizeFilename 清理文件名，移除不安全的字符
func sanitizeFilename(filename string) string {
	// 替换不安全的字符
	safe := filepath.Clean(filename)
	safe = filepath.Base(safe)

	// 确保文件名不为空
	if safe == "" || safe == "." {
		return "unnamed_file"
	}

	return safe
}
