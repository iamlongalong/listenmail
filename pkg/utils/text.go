package utils

import (
	"strings"

	"github.com/iamlongalong/listenmail/pkg/types"
	"github.com/jaytaylor/html2text"
)

// ToPlainText 将邮件内容转换为纯文本
// 如果邮件包含纯文本内容，则直接返回
// 如果只有HTML内容，则提取HTML中的文本
func ToPlainText(mail *types.Mail) string {
	// 如果有纯文本内容，优先使用
	if mail.Text != "" {
		return cleanText(mail.Text)
	}

	// 如果有HTML内容，提取文本
	if mail.HTML != "" {
		text, err := html2text.FromString(mail.HTML, html2text.Options{
			PrettyTables:        true,                               // 使用 ASCII 表格
			OmitLinks:           false,                              // 保留链接URL
			TextOnly:            false,                              // 保留基本格式
			PrettyTablesOptions: html2text.NewPrettyTablesOptions(), // 使用默认的表格选项
		})
		if err != nil {
			return ""
		}
		return cleanText(text)
	}

	return ""
}

// cleanText 清理提取的文本
func cleanText(text string) string {
	// 将多个空白字符替换为单个空格
	text = strings.Join(strings.Fields(text), " ")

	// 将多个换行符替换为双换行符
	lines := strings.Split(text, "\n")
	var cleanLines []string
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			cleanLines = append(cleanLines, strings.TrimSpace(line))
		}
	}
	text = strings.Join(cleanLines, "\n\n")

	return strings.TrimSpace(text)
}

// GetPreview 获取邮件内容的预览（前N个字符）
func GetPreview(mail *types.Mail, length int) string {
	text := ToPlainText(mail)
	if len(text) <= length {
		return text
	}
	return text[:length] + "..."
}
