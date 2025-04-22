package types

import (
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
