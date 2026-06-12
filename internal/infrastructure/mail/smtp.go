package mailinfra

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/smtp"

	"github.com/jordan-wright/email"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/config"
	appmail "github.com/rifkifajarramadhani/golang-clean-architecture/internal/mail"
)

type SMTPTransport struct {
	cfg config.MailConfig
}

func NewSMTPTransport(cfg config.MailConfig) *SMTPTransport {
	return &SMTPTransport{cfg: cfg}
}

func (t *SMTPTransport) Send(ctx context.Context, message appmail.Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	outgoing := email.NewEmail()
	outgoing.From = message.Envelope.From.String()
	outgoing.Subject = message.Envelope.Subject
	outgoing.Text = []byte(message.Content.Text)
	outgoing.HTML = []byte(message.Content.HTML)
	for _, recipient := range message.Envelope.To {
		outgoing.To = append(outgoing.To, recipient.String())
	}
	for _, attachment := range message.Attachments {
		contentType := attachment.ContentType
		if contentType == "" {
			contentType = "application/octet-stream"
		}
		if _, err := outgoing.Attach(bytes.NewReader(attachment.Data), attachment.Filename, contentType); err != nil {
			return fmt.Errorf("attach %q: %w", attachment.Filename, err)
		}
	}

	address := net.JoinHostPort(t.cfg.Host, fmt.Sprintf("%d", t.cfg.Port))
	var auth smtp.Auth
	if t.cfg.Username != "" {
		auth = smtp.PlainAuth("", t.cfg.Username, t.cfg.Password, t.cfg.Host)
	}
	tlsConfig := &tls.Config{ServerName: t.cfg.Host, MinVersion: tls.VersionTLS12}

	switch t.cfg.Encryption {
	case config.MailEncryptionTLS:
		return outgoing.SendWithTLS(address, auth, tlsConfig)
	case config.MailEncryptionStartTLS:
		return outgoing.SendWithStartTLS(address, auth, tlsConfig)
	default:
		return outgoing.Send(address, auth)
	}
}
