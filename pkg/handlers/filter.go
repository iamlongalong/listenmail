package handlers

import (
	"regexp"
	"time"

	"github.com/iamlongalong/listenmail/pkg/types"
)

// Condition 定义邮件匹配条件
type Condition func(*types.Mail) bool

// Handler 是一个基于条件的邮件处理器
type Handler struct {
	handle    func(*types.Mail) error
	condition Condition
}

// NewHandler 创建一个新的处理器
func NewHandler(handle func(*types.Mail) error, condition Condition) *Handler {
	if condition == nil {
		condition = func(*types.Mail) bool { return true }
	}
	return &Handler{
		handle:    handle,
		condition: condition,
	}
}

// Handle 实现 Handler 接口
func (h *Handler) Handle(mail *types.Mail) error {
	return h.handle(mail)
}

// Match 实现 Handler 接口
func (h *Handler) Match(mail *types.Mail) bool {
	return h.condition(mail)
}

// 条件组合器

// And 组合多个条件，所有条件都必须满足
func And(conditions ...Condition) Condition {
	return func(m *types.Mail) bool {
		for _, c := range conditions {
			if !c(m) {
				return false
			}
		}
		return true
	}
}

// Or 组合多个条件，满足任意条件即可
func Or(conditions ...Condition) Condition {
	return func(m *types.Mail) bool {
		for _, c := range conditions {
			if c(m) {
				return true
			}
		}
		return false
	}
}

// Not 对条件取反
func Not(condition Condition) Condition {
	return func(m *types.Mail) bool {
		return !condition(m)
	}
}

// 预定义条件构造器

// From 创建发件人匹配条件
func From(pattern string) Condition {
	re := regexp.MustCompile(pattern)
	return func(m *types.Mail) bool {
		for _, addr := range m.From {
			if re.MatchString(addr.Address) {
				return true
			}
		}
		return false
	}
}

// To 创建收件人匹配条件
func To(pattern string) Condition {
	re := regexp.MustCompile(pattern)
	return func(m *types.Mail) bool {
		for _, addr := range m.To {
			if re.MatchString(addr.Address) {
				return true
			}
		}
		return false
	}
}

// Cc 创建抄送人匹配条件
func Cc(pattern string) Condition {
	re := regexp.MustCompile(pattern)
	return func(m *types.Mail) bool {
		for _, addr := range m.Cc {
			if re.MatchString(addr.Address) {
				return true
			}
		}
		return false
	}
}

// Subject 创建主题匹配条件
func Subject(pattern string) Condition {
	re := regexp.MustCompile(pattern)
	return func(m *types.Mail) bool {
		return re.MatchString(m.Subject)
	}
}

// DateAfter 创建日期晚于指定时间的条件
func DateAfter(t time.Time) Condition {
	return func(m *types.Mail) bool {
		return m.Date.After(t)
	}
}

// DateBefore 创建日期早于指定时间的条件
func DateBefore(t time.Time) Condition {
	return func(m *types.Mail) bool {
		return m.Date.Before(t)
	}
}

// HasAttachment 创建具有附件的条件
func HasAttachment() Condition {
	return func(m *types.Mail) bool {
		return len(m.Attachments) > 0
	}
}

// AttachmentName 创建附件名称匹配条件
func AttachmentName(pattern string) Condition {
	re := regexp.MustCompile(pattern)
	return func(m *types.Mail) bool {
		for _, att := range m.Attachments {
			if re.MatchString(att.Filename) {
				return true
			}
		}
		return false
	}
}

// Header 创建邮件头匹配条件
func Header(name, pattern string) Condition {
	re := regexp.MustCompile(pattern)
	return func(m *types.Mail) bool {
		if values, ok := m.Headers[name]; ok {
			for _, v := range values {
				if re.MatchString(v) {
					return true
				}
			}
		}
		return false
	}
}

// TextContent 创建文本内容匹配条件
func TextContent(pattern string) Condition {
	re := regexp.MustCompile(pattern)
	return func(m *types.Mail) bool {
		return re.MatchString(m.Text)
	}
}

// HTMLContent 创建HTML内容匹配条件
func HTMLContent(pattern string) Condition {
	re := regexp.MustCompile(pattern)
	return func(m *types.Mail) bool {
		return re.MatchString(m.HTML)
	}
}

// AnyContent 创建任意内容（文本或HTML）匹配条件
func AnyContent(pattern string) Condition {
	re := regexp.MustCompile(pattern)
	return func(m *types.Mail) bool {
		return re.MatchString(m.Text) || re.MatchString(m.HTML)
	}
}
