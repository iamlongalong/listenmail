package types

import (
	"fmt"
	"strings"
	"time"

	"github.com/emersion/go-message/mail"
)

// Mail represents a unified mail message structure
type Mail struct {
	ID          string
	From        []*mail.Address
	To          []*mail.Address
	Cc          []*mail.Address
	Bcc         []*mail.Address
	ReplyTo     []*mail.Address
	Subject     string
	Date        time.Time
	MessageID   string
	InReplyTo   []string
	References  []string
	Text        string // 纯文本内容
	HTML        string // HTML内容
	Attachments []Attachment
	Headers     map[string][]string

	Source string
}

// Attachment represents an email attachment
type Attachment struct {
	Filename    string
	ContentType string
	Data        []byte
	Header      mail.Header
}

// Source represents a mail source interface
type Source interface {
	// Start starts the mail source
	Start() error
	// Stop stops the mail source
	Stop() error
	// Name returns the source name
	Name() string
}

// Handler represents a mail processing handler
type Handler interface {
	// Handle processes a mail message
	Handle(mail *Mail) error
	// Match checks if this handler should process the mail
	Match(mail *Mail) bool
}

// Dispatcher manages mail handlers and dispatches mails to matching handlers
type Dispatcher interface {
	// AddHandler adds a new mail handler
	AddHandlers(handlers ...Handler) error
	// RemoveHandler removes a mail handler
	RemoveHandlers(handlers ...Handler) error
	// Dispatch dispatches a mail to matching handlers
	Dispatch(mail *Mail) error
}

type ConfigFile struct {
	Server struct {
		Addr string `yaml:"addr"`

		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"server"`
	Save struct {
		Dir string `yaml:"dir"`
	} `yaml:"save"`

	Sources struct {
		// 各个源的具体配置
		SMTP    []*SMTPConfig    `yaml:"smtp,omitempty"`
		IMAP    []*IMAPConfig    `yaml:"imap,omitempty"`
		POP3    []*POP3Config    `yaml:"pop3,omitempty"`
		MailHog []*MailHogConfig `yaml:"mailhog,omitempty"`
	} `yaml:"sources"`
}

// SMTPConfig represents SMTP server configuration
type SMTPConfig struct {
	Name    string `yaml:"name"`
	Enabled bool   `yaml:"enabled"`

	Address           string        `yaml:"address"`
	Domain            string        `yaml:"domain"`
	ReadTimeout       time.Duration `yaml:"read_timeout"`
	WriteTimeout      time.Duration `yaml:"write_timeout"`
	MaxMessageBytes   int64         `yaml:"max_message_bytes"`
	MaxRecipients     int           `yaml:"max_recipients"`
	AllowInsecureAuth bool          `yaml:"allow_insecure_auth"`
}

// IMAPConfig represents IMAP client configuration
type IMAPConfig struct {
	Name    string `yaml:"name"`
	Enabled bool   `yaml:"enabled"`

	Server   string        `yaml:"server"`
	Username string        `yaml:"username"`
	Password string        `yaml:"password"`
	TLS      bool          `yaml:"tls"`
	Interval time.Duration `yaml:"check_interval"`
}

// POP3Config represents POP3 client configuration
type POP3Config struct {
	Name    string `yaml:"name"`
	Enabled bool   `yaml:"enabled"`

	Server   string        `yaml:"server"`
	Username string        `yaml:"username"`
	Password string        `yaml:"password"`
	TLS      bool          `yaml:"tls"`
	Interval time.Duration `yaml:"check_interval"`
}

// MailHogConfig represents MailHog API client configuration
type MailHogConfig struct {
	Name    string `yaml:"name"`
	Enabled bool   `yaml:"enabled"`

	APIURL   string        `yaml:"api_url"`
	Interval time.Duration `yaml:"check_interval"`
}

// SourceType represents the type of mail source
type SourceType string

const (
	SourceTypeSMTP    SourceType = "smtp"
	SourceTypeIMAP    SourceType = "imap"
	SourceTypePOP3    SourceType = "pop3"
	SourceTypeMailHog SourceType = "mailhog"
)

// APIAddress represents an email address in API responses
type APIAddress struct {
	Address string `json:"address"`
	Name    string `json:"name"`
}

// APIAttachment represents a mail attachment in API responses
type APIAttachment struct {
	ID          int64  `json:"id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
	Path        string `json:"path"`
}

// APIMail represents a mail record in API responses
type APIMail struct {
	ID                      int64           `json:"id"`
	MessageID               string          `json:"message_id"`
	Subject                 string          `json:"subject"`
	Date                    time.Time       `json:"date"`
	TextContent             string          `json:"text_content"`
	HTMLContent             string          `json:"html_content"`
	ContentType             string          `json:"content_type"`
	ContentTransferEncoding string          `json:"content_transfer_encoding"`
	ReplyTo                 string          `json:"reply_to"`
	InReplyTo               string          `json:"in_reply_to"`
	References              string          `json:"references"`
	Priority                string          `json:"priority"`
	XPriority               string          `json:"x_priority"`
	Importance              string          `json:"importance"`
	RawHeaders              string          `json:"raw_headers"`
	CreatedAt               time.Time       `json:"created_at"`
	From                    []APIAddress    `json:"from"`
	To                      []APIAddress    `json:"to"`
	Cc                      []APIAddress    `json:"cc"`
	Bcc                     []APIAddress    `json:"bcc"`
	Attachments             []APIAttachment `json:"attachments"`
	Source                  string          `json:"source"`
}

// ToAPIAddress converts a mail.Address to an APIAddress
func ToAPIAddress(addr *mail.Address) APIAddress {
	if addr == nil {
		return APIAddress{}
	}
	return APIAddress{
		Address: addr.Address,
		Name:    addr.Name,
	}
}

// ToAPIAddresses converts a slice of mail.Address to APIAddress
func ToAPIAddresses(addrs []*mail.Address) []APIAddress {
	result := make([]APIAddress, 0, len(addrs))
	for _, addr := range addrs {
		result = append(result, ToAPIAddress(addr))
	}
	return result
}

// ToAPIMail converts a Mail to an APIMail
func (m *Mail) ToAPIMail() *APIMail {
	api := &APIMail{
		MessageID:   m.MessageID,
		Subject:     m.Subject,
		Date:        m.Date,
		TextContent: m.Text,
		HTMLContent: m.HTML,
		From:        ToAPIAddresses(m.From),
		To:          ToAPIAddresses(m.To),
		Cc:          ToAPIAddresses(m.Cc),
		Bcc:         ToAPIAddresses(m.Bcc),
	}

	// Convert headers
	if values := m.Headers["Content-Type"]; len(values) > 0 {
		api.ContentType = values[0]
	}
	if values := m.Headers["Content-Transfer-Encoding"]; len(values) > 0 {
		api.ContentTransferEncoding = values[0]
	}
	if len(m.ReplyTo) > 0 {
		api.ReplyTo = m.ReplyTo[0].String()
	}
	if len(m.InReplyTo) > 0 {
		api.InReplyTo = m.InReplyTo[0]
	}
	if len(m.References) > 0 {
		api.References = strings.Join(m.References, " ")
	}
	if values := m.Headers["Priority"]; len(values) > 0 {
		api.Priority = values[0]
	}
	if values := m.Headers["X-Priority"]; len(values) > 0 {
		api.XPriority = values[0]
	}
	if values := m.Headers["Importance"]; len(values) > 0 {
		api.Importance = values[0]
	}

	// Build raw headers
	var rawHeaders strings.Builder
	for name, values := range m.Headers {
		switch strings.ToLower(name) {
		case "content-type", "content-transfer-encoding", "reply-to",
			"in-reply-to", "references", "priority", "x-priority", "importance":
			continue
		default:
			for _, value := range values {
				if rawHeaders.Len() > 0 {
					rawHeaders.WriteString("\n")
				}
				rawHeaders.WriteString(fmt.Sprintf("%s: %s", name, value))
			}
		}
	}
	api.RawHeaders = rawHeaders.String()

	return api
}
