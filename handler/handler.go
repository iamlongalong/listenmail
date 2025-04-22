package handler

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/iamlongalong/listenmail/pkg/handlers"
	"github.com/iamlongalong/listenmail/pkg/types"
	"github.com/iamlongalong/listenmail/pkg/utils"
)

// SaveHandler 创建一个保存所有邮件到数据库的处理器
func SaveHandler(dir string) types.Handler {
	// 确保数据目录存在
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("Error creating data directory: %v", err)
		return nil
	}

	// 创建保存处理器
	handler, err := handlers.NewSaveHandler(handlers.SaveConfig{
		DBPath:        filepath.Join(dir, "emails.db"),
		AttachmentDir: filepath.Join(dir, "attachments"),
	})
	if err != nil {
		log.Printf("Error creating save handler: %v", err)
		return nil
	}

	return handler
}

// CursorCodeHandler 创建一个处理 Cursor 相关邮件的处理器
func CursorCodeHandler() types.Handler {
	return handlers.NewHandler(
		// 处理函数
		func(mail *types.Mail) error {
			// 提取纯文本内容
			content := utils.ToPlainText(mail)

			// 查找并提取代码块
			codeBlocks := extractCodeBlocks(content)
			if len(codeBlocks) > 0 {
				log.Printf("Found %d code blocks in email: %s", len(codeBlocks), mail.Subject)
				for i, code := range codeBlocks {
					log.Printf("Code block %d:\n%s\n", i+1, code)
				}
			}

			return nil
		},
		// 匹配条件：组合多个条件
		handlers.And(
			// 发件人是 Cursor 相关的邮件地址
			handlers.Or(
				handlers.From(".*@cursor.so"),
				handlers.From(".*@cursor.sh"),
				handlers.Subject(".*\\[Cursor\\].*"),
			),
			// 确保邮件内容不为空
			func(m *types.Mail) bool {
				return m.Text != "" || m.HTML != ""
			},
		),
	)
}

// extractCodeBlocks 从文本中提取代码块
func extractCodeBlocks(content string) []string {
	var blocks []string

	// 匹配 Markdown 风格的代码块
	// ```language
	// code
	// ```
	markdownPattern := regexp.MustCompile("```(?:[a-zA-Z0-9]+)?\n([\\s\\S]*?)```")
	matches := markdownPattern.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		if len(match) > 1 {
			// 清理代码块
			code := strings.TrimSpace(match[1])
			if code != "" {
				blocks = append(blocks, code)
			}
		}
	}

	// 如果没有找到 Markdown 风格的代码块，尝试查找缩进代码块
	if len(blocks) == 0 {
		lines := strings.Split(content, "\n")
		var currentBlock strings.Builder
		inBlock := false

		for _, line := range lines {
			// 检查是否是缩进的代码行（4个空格或1个制表符）
			if strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t") {
				if !inBlock {
					inBlock = true
				}
				// 移除缩进
				code := strings.TrimPrefix(strings.TrimPrefix(line, "    "), "\t")
				currentBlock.WriteString(code)
				currentBlock.WriteString("\n")
			} else if inBlock {
				// 代码块结束
				block := strings.TrimSpace(currentBlock.String())
				if block != "" {
					blocks = append(blocks, block)
				}
				currentBlock.Reset()
				inBlock = false
			}
		}

		// 检查最后一个代码块
		if inBlock {
			block := strings.TrimSpace(currentBlock.String())
			if block != "" {
				blocks = append(blocks, block)
			}
		}
	}

	return blocks
}

// 可选：添加代码块保存功能
type CodeBlock struct {
	Language string
	Content  string
	LineNum  int
}

// parseCodeBlock 解析代码块，识别语言和行号
func parseCodeBlock(raw string) CodeBlock {
	lines := strings.Split(raw, "\n")
	if len(lines) == 0 {
		return CodeBlock{}
	}

	// 尝试从第一行识别语言和行号
	// 例如：```python:10 或 ```java:20-30
	firstLine := lines[0]
	if strings.HasPrefix(firstLine, "```") {
		parts := strings.Split(strings.TrimPrefix(firstLine, "```"), ":")
		if len(parts) > 1 {
			return CodeBlock{
				Language: parts[0],
				Content:  strings.Join(lines[1:], "\n"),
				LineNum:  parseLineNum(parts[1]),
			}
		}
		return CodeBlock{
			Language: parts[0],
			Content:  strings.Join(lines[1:], "\n"),
		}
	}

	// 如果没有语言标记，返回原始内容
	return CodeBlock{
		Content: raw,
	}
}

// parseLineNum 解析行号信息
func parseLineNum(s string) int {
	// 移除任何非数字字符
	numStr := regexp.MustCompile(`\d+`).FindString(s)
	if numStr == "" {
		return 0
	}
	num := 0
	_, err := fmt.Sscanf(numStr, "%d", &num)
	if err != nil {
		return 0
	}
	return num
}
