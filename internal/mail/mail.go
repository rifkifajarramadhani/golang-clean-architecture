package mail

import (
	"context"
	"errors"
	"fmt"
	stdmail "net/mail"
	"strings"
	"time"
)

const TypeSend = "mail:send"

type Address struct {
	Name    string `json:"name,omitempty"`
	Address string `json:"address"`
}

func (a Address) String() string {
	return (&stdmail.Address{Name: a.Name, Address: a.Address}).String()
}

type Envelope struct {
	From    Address   `json:"from"`
	To      []Address `json:"to"`
	Subject string    `json:"subject"`
}

type Content struct {
	Text string `json:"text,omitempty"`
	HTML string `json:"html,omitempty"`
}

type Attachment struct {
	Filename    string `json:"filename"`
	ContentType string `json:"content_type,omitempty"`
	Data        []byte `json:"data"`
}

type Message struct {
	Envelope    Envelope     `json:"envelope"`
	Content     Content      `json:"content"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

type Mailable interface {
	Envelope() Envelope
	Content() (Content, error)
	Attachments() []Attachment
}

type Transport interface {
	Send(context.Context, Message) error
}

type QueueOptions struct {
	Queue     string
	ProcessAt time.Time
	MaxRetry  int
	Timeout   time.Duration
	UniqueFor time.Duration
	Retention time.Duration
	TaskID    string
}

type QueuedMessageInfo struct {
	ID    string
	Queue string
}

type QueuedMessageDispatcher interface {
	DispatchMessage(context.Context, SendJob, QueueOptions) (*QueuedMessageInfo, error)
}

type Mailer struct {
	defaultFrom Address
	transport   Transport
	dispatcher  QueuedMessageDispatcher
}

func NewMailer(defaultFrom Address, transport Transport, dispatcher QueuedMessageDispatcher) *Mailer {
	return &Mailer{defaultFrom: defaultFrom, transport: transport, dispatcher: dispatcher}
}

func (m *Mailer) Render(mailable Mailable) (Message, error) {
	if mailable == nil {
		return Message{}, errors.New("mailable is required")
	}
	envelope := mailable.Envelope()
	if strings.TrimSpace(envelope.From.Address) == "" {
		envelope.From = m.defaultFrom
	}
	content, err := mailable.Content()
	if err != nil {
		return Message{}, fmt.Errorf("render mail content: %w", err)
	}
	message := Message{
		Envelope:    envelope,
		Content:     content,
		Attachments: mailable.Attachments(),
	}
	if err := validateMessage(message); err != nil {
		return Message{}, err
	}
	return message, nil
}

func (m *Mailer) Send(ctx context.Context, mailable Mailable) error {
	if m.transport == nil {
		return errors.New("mail transport is not configured")
	}
	message, err := m.Render(mailable)
	if err != nil {
		return err
	}
	return m.transport.Send(ctx, message)
}

func (m *Mailer) Queue(ctx context.Context, mailable Mailable, options QueueOptions) (*QueuedMessageInfo, error) {
	if m.dispatcher == nil {
		return nil, errors.New("queue dispatcher is not configured")
	}
	message, err := m.Render(mailable)
	if err != nil {
		return nil, err
	}
	if options.Queue == "" {
		options.Queue = "mail"
	}
	if options.MaxRetry <= 0 {
		options.MaxRetry = 3
	}
	if options.Timeout <= 0 {
		options.Timeout = 30 * time.Second
	}
	return m.dispatcher.DispatchMessage(ctx, SendJob{Message: message}, options)
}

type SendJob struct {
	Message Message `json:"message"`
}

func validateMessage(message Message) error {
	if _, err := stdmail.ParseAddress(message.Envelope.From.Address); err != nil {
		return fmt.Errorf("invalid from address: %w", err)
	}
	if len(message.Envelope.To) == 0 {
		return errors.New("at least one recipient is required")
	}
	for _, recipient := range message.Envelope.To {
		if _, err := stdmail.ParseAddress(recipient.Address); err != nil {
			return fmt.Errorf("invalid recipient address: %w", err)
		}
	}
	if strings.TrimSpace(message.Envelope.Subject) == "" {
		return errors.New("mail subject is required")
	}
	if message.Content.Text == "" && message.Content.HTML == "" {
		return errors.New("mail content is required")
	}
	for _, attachment := range message.Attachments {
		if strings.TrimSpace(attachment.Filename) == "" {
			return errors.New("attachment filename is required")
		}
	}
	return nil
}
