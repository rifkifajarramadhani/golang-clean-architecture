package smtp

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"

	gomail "github.com/wneessen/go-mail"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/config"
	appmail "github.com/rifkifajarramadhani/golang-clean-architecture/internal/mail"
)

type Transport struct {
	client *gomail.Client
}

func NewTransport(cfg config.MailConfig) (*Transport, error) {
	options := []gomail.Option{
		gomail.WithPort(cfg.Port),
		gomail.WithTLSConfig(&tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12}),
	}
	switch cfg.Encryption {
	case config.MailEncryptionTLS:
		options = append(options, gomail.WithSSL(), gomail.WithTLSPolicy(gomail.TLSMandatory))
	case config.MailEncryptionStartTLS:
		options = append(options, gomail.WithTLSPolicy(gomail.TLSMandatory))
	default:
		options = append(options, gomail.WithTLSPolicy(gomail.NoTLS))
	}
	if cfg.Username != "" {
		options = append(options,
			gomail.WithSMTPAuth(gomail.SMTPAuthAutoDiscover),
			gomail.WithUsername(cfg.Username),
			gomail.WithPassword(cfg.Password),
		)
	}
	client, err := gomail.NewClient(cfg.Host, options...)
	if err != nil {
		return nil, fmt.Errorf("create SMTP client: %w", err)
	}
	return &Transport{client: client}, nil
}

func (t *Transport) Send(ctx context.Context, message appmail.Message) error {
	outgoing := gomail.NewMsg()
	if err := outgoing.FromFormat(message.Envelope.From.Name, message.Envelope.From.Address); err != nil {
		return fmt.Errorf("set sender: %w", err)
	}
	for _, recipient := range message.Envelope.To {
		if err := outgoing.AddToFormat(recipient.Name, recipient.Address); err != nil {
			return fmt.Errorf("set recipient: %w", err)
		}
	}
	outgoing.Subject(message.Envelope.Subject)
	if message.Content.Text != "" {
		outgoing.SetBodyString(gomail.TypeTextPlain, message.Content.Text)
	}
	if message.Content.HTML != "" {
		if message.Content.Text == "" {
			outgoing.SetBodyString(gomail.TypeTextHTML, message.Content.HTML)
		} else {
			outgoing.AddAlternativeString(gomail.TypeTextHTML, message.Content.HTML)
		}
	}
	for _, attachment := range message.Attachments {
		options := []gomail.FileOption{}
		if attachment.ContentType != "" {
			options = append(options, gomail.WithFileContentType(gomail.ContentType(attachment.ContentType)))
		}
		if err := outgoing.AttachReader(attachment.Filename, bytes.NewReader(attachment.Data), options...); err != nil {
			return fmt.Errorf("attach %q: %w", attachment.Filename, err)
		}
	}
	return t.client.DialAndSendWithContext(ctx, outgoing)
}

var _ appmail.Transport = (*Transport)(nil)
