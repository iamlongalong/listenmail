package sources

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/emersion/go-smtp"
	"github.com/iamlongalong/listenmail/pkg/types"
	"github.com/iamlongalong/listenmail/pkg/utils"
)

// SMTPSource implements a SMTP server as a mail source
type SMTPSource struct {
	config     *types.SMTPConfig
	server     *smtp.Server
	dispatcher types.Dispatcher
}

// NewSMTPSource creates a new SMTP source
func NewSMTPSource(config *types.SMTPConfig, dispatcher types.Dispatcher) (*SMTPSource, error) {
	s := &SMTPSource{
		config:     config,
		dispatcher: dispatcher,
	}

	if config == nil {
		config = &types.SMTPConfig{
			Address:           ":25",
			Domain:            "localhost",
			ReadTimeout:       10 * time.Second,
			WriteTimeout:      10 * time.Second,
			MaxMessageBytes:   10 * 1024 * 1024, // 10MB
			MaxRecipients:     50,
			AllowInsecureAuth: true,
		}
	}

	backend := &Backend{
		dispatcher: dispatcher,
	}

	s.server = smtp.NewServer(backend)
	s.server.Addr = config.Address
	s.server.Domain = config.Domain
	s.server.ReadTimeout = config.ReadTimeout
	s.server.WriteTimeout = config.WriteTimeout
	s.server.MaxMessageBytes = config.MaxMessageBytes
	s.server.MaxRecipients = config.MaxRecipients
	s.server.AllowInsecureAuth = config.AllowInsecureAuth

	return s, nil
}

// Start implements Source interface
func (s *SMTPSource) Start() error {
	log.Println("smtp source is running...")
	go func() {
		log.Fatal(s.server.ListenAndServe())
	}()
	return nil
}

// Stop implements Source interface
func (s *SMTPSource) Stop() error {
	log.Println("smtp source is stopping...")
	return s.server.Close()
}

// Name implements Source interface
func (s *SMTPSource) Name() string {
	return s.config.Name
}

// Backend implements SMTP server methods
type Backend struct {
	dispatcher types.Dispatcher
}

func (bkd *Backend) NewSession(_ *smtp.Conn) (smtp.Session, error) {
	return &Session{
		dispatcher: bkd.dispatcher,
	}, nil
}

// Session implements SMTP session
type Session struct {
	dispatcher types.Dispatcher
	from       string
	to         []string
}

func (s *Session) AuthPlain(username, password string) error {
	return nil
}

func (s *Session) Mail(from string, opts *smtp.MailOptions) error {
	s.from = from
	return nil
}

func (s *Session) Rcpt(to string, opts *smtp.RcptOptions) error {
	s.to = append(s.to, to)
	return nil
}

func (s *Session) Data(r io.Reader) error {
	// 读取邮件内容
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, r); err != nil {
		return err
	}

	// 解析邮件
	mail, err := utils.ParseMail(buf)
	if err != nil {
		return err
	}
	if mail.ID == "" {
		mail.ID = mail.MessageID
	}
	if mail.ID == "" {
		mail.ID = fmt.Sprintf("%s:%s:%s", mail.Date, mail.From[0].String(), mail.To[0].String())
	}

	// 分发邮件
	return s.dispatcher.Dispatch(mail)
}

func (s *Session) Reset() {
	s.from = ""
	s.to = nil
}

func (s *Session) Logout() error {
	return nil
}
