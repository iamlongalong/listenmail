package types

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

// DBMail represents the mail record in database
type DBMail struct {
	gorm.Model
	MessageID               string    `gorm:"index;type:text"`
	Subject                 string    `gorm:"type:text"`
	Date                    time.Time `gorm:"index"`
	TextContent             string    `gorm:"type:text"`
	HTMLContent             string    `gorm:"type:text"`
	ContentType             string    `gorm:"type:text"`
	ContentTransferEncoding string    `gorm:"type:text"`
	ReplyTo                 string    `gorm:"type:text"`
	InReplyTo               string    `gorm:"type:text"`
	References              string    `gorm:"type:text"`
	Priority                string    `gorm:"type:text"`
	XPriority               string    `gorm:"type:text"`
	Importance              string    `gorm:"type:text"`
	RawHeaders              string    `gorm:"type:text"`

	// Relations
	From        []DBAddress    `gorm:"foreignKey:MailID;constraint:OnDelete:CASCADE"`
	To          []DBAddress    `gorm:"foreignKey:MailID;constraint:OnDelete:CASCADE"`
	Cc          []DBAddress    `gorm:"foreignKey:MailID;constraint:OnDelete:CASCADE"`
	Bcc         []DBAddress    `gorm:"foreignKey:MailID;constraint:OnDelete:CASCADE"`
	Attachments []DBAttachment `gorm:"foreignKey:MailID;constraint:OnDelete:CASCADE"`

	// Source
	Source string `gorm:"type:text"`
}

// DBAddress represents an email address in database
type DBAddress struct {
	gorm.Model
	MailID  uint   `gorm:"index"`
	Type    string `gorm:"type:text;index"` // from, to, cc, bcc
	Address string `gorm:"type:text"`
	Name    string `gorm:"type:text"`
}

// DBAttachment represents a mail attachment in database
type DBAttachment struct {
	gorm.Model
	MailID      uint   `gorm:"index"`
	Filename    string `gorm:"type:text"`
	ContentType string `gorm:"type:text"`
	Size        int64
	Path        string `gorm:"type:text"`
}

// ToAPIMail converts DBMail to APIMail
func (m *DBMail) ToAPIMail() *APIMail {
	api := &APIMail{
		ID:                      int64(m.ID),
		MessageID:               m.MessageID,
		Subject:                 m.Subject,
		Date:                    m.Date,
		TextContent:             m.TextContent,
		HTMLContent:             m.HTMLContent,
		ContentType:             m.ContentType,
		ContentTransferEncoding: m.ContentTransferEncoding,
		ReplyTo:                 m.ReplyTo,
		InReplyTo:               m.InReplyTo,
		References:              m.References,
		Priority:                m.Priority,
		XPriority:               m.XPriority,
		Importance:              m.Importance,
		RawHeaders:              m.RawHeaders,
		CreatedAt:               m.CreatedAt,
		Source:                  m.Source,
	}

	// Convert addresses
	for _, addr := range m.From {
		api.From = append(api.From, APIAddress{
			Address: addr.Address,
			Name:    addr.Name,
		})
	}
	for _, addr := range m.To {
		api.To = append(api.To, APIAddress{
			Address: addr.Address,
			Name:    addr.Name,
		})
	}
	for _, addr := range m.Cc {
		api.Cc = append(api.Cc, APIAddress{
			Address: addr.Address,
			Name:    addr.Name,
		})
	}
	for _, addr := range m.Bcc {
		api.Bcc = append(api.Bcc, APIAddress{
			Address: addr.Address,
			Name:    addr.Name,
		})
	}

	// Convert attachments
	for _, att := range m.Attachments {
		api.Attachments = append(api.Attachments, APIAttachment{
			ID:          int64(att.ID),
			Filename:    att.Filename,
			ContentType: att.ContentType,
			Size:        att.Size,
			Path:        att.Path,
		})
	}

	return api
}

// FromMail converts Mail to DBMail
func FromMail(m *Mail) *DBMail {
	dbMail := &DBMail{
		MessageID:               m.MessageID,
		Subject:                 m.Subject,
		Date:                    m.Date,
		TextContent:             m.Text,
		HTMLContent:             m.HTML,
		Source:                  m.Source,
		ContentTransferEncoding: getFirstHeader(m.Headers, "Content-Transfer-Encoding"),
		ContentType:             getFirstHeader(m.Headers, "Content-Type"),
		Priority:                getFirstHeader(m.Headers, "Priority"),
		XPriority:               getFirstHeader(m.Headers, "X-Priority"),
		Importance:              getFirstHeader(m.Headers, "Importance"),
	}

	// Convert ReplyTo
	if len(m.ReplyTo) > 0 {
		dbMail.ReplyTo = m.ReplyTo[0].String()
	}

	// Convert InReplyTo
	if len(m.InReplyTo) > 0 {
		dbMail.InReplyTo = m.InReplyTo[0]
	}

	// Convert References
	if len(m.References) > 0 {
		dbMail.References = m.References[0]
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
	dbMail.RawHeaders = rawHeaders.String()

	// Convert addresses
	for _, addr := range m.From {
		dbMail.From = append(dbMail.From, DBAddress{
			Type:    "from",
			Address: addr.Address,
			Name:    addr.Name,
		})
	}
	for _, addr := range m.To {
		dbMail.To = append(dbMail.To, DBAddress{
			Type:    "to",
			Address: addr.Address,
			Name:    addr.Name,
		})
	}
	for _, addr := range m.Cc {
		dbMail.Cc = append(dbMail.Cc, DBAddress{
			Type:    "cc",
			Address: addr.Address,
			Name:    addr.Name,
		})
	}
	for _, addr := range m.Bcc {
		dbMail.Bcc = append(dbMail.Bcc, DBAddress{
			Type:    "bcc",
			Address: addr.Address,
			Name:    addr.Name,
		})
	}

	return dbMail
}

func getFirstHeader(headers map[string][]string, key string) string {
	if values := headers[key]; len(values) > 0 {
		return values[0]
	}
	return ""
}
