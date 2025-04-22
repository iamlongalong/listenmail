package utils

import (
	"bytes"
	"io"
	"io/ioutil"
	"strings"

	"github.com/emersion/go-message/mail"
	"github.com/iamlongalong/listenmail/pkg/types"
)

// ParseMail 将邮件消息解析为Mail结构体
func ParseMail(r io.Reader) (*types.Mail, error) {
	// 解析邮件消息
	mr, err := mail.CreateReader(r)
	if err != nil {
		return nil, err
	}

	// 创建Mail结构体
	m := &types.Mail{
		Headers: make(map[string][]string),
	}

	// 获取头部信息
	header := mr.Header
	m.From, _ = header.AddressList("From")
	m.To, _ = header.AddressList("To")
	m.Cc, _ = header.AddressList("Cc")
	m.Bcc, _ = header.AddressList("Bcc")
	m.ReplyTo, _ = header.AddressList("Reply-To")
	m.Subject, _ = header.Subject()
	m.MessageID = header.Get("Message-ID")
	m.Date, _ = header.Date()

	// 获取References和In-Reply-To
	if refs := header.Get("References"); refs != "" {
		m.References = strings.Fields(refs)
	}
	if inReplyTo := header.Get("In-Reply-To"); inReplyTo != "" {
		m.InReplyTo = strings.Fields(inReplyTo)
	}

	// 复制常见的头部信息
	commonHeaders := []string{
		"From", "To", "Cc", "Bcc", "Subject", "Date", "Message-ID",
		"References", "In-Reply-To", "Reply-To", "Content-Type",
		"Content-Transfer-Encoding", "MIME-Version",
	}
	for _, key := range commonHeaders {
		if values := header.Values(key); len(values) > 0 {
			m.Headers[key] = values
		}
	}

	// 处理邮件正文和附件
	for {
		p, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		switch h := p.Header.(type) {
		case *mail.InlineHeader:
			// 处理正文部分
			contentType, _, _ := h.ContentType()
			content, _ := ioutil.ReadAll(p.Body)

			switch contentType {
			case "text/plain":
				m.Text = string(content)
			case "text/html":
				m.HTML = string(content)
			}

		case *mail.AttachmentHeader:
			// 处理附件
			filename, _ := h.Filename()
			contentType, _, _ := h.ContentType()
			content, _ := ioutil.ReadAll(p.Body)

			// 创建附件
			att := types.Attachment{
				Filename:    filename,
				ContentType: contentType,
				Data:        content,
			}

			m.Attachments = append(m.Attachments, att)
		}
	}

	return m, nil
}

// CreateMailReader 从原始邮件数据创建邮件读取器
func CreateMailReader(data []byte) (io.Reader, error) {
	return bytes.NewReader(data), nil
}
