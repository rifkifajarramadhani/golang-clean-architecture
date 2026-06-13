package mail

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

type mailableFake struct {
	envelope    Envelope
	content     Content
	attachments []Attachment
	err         error
}

func (m mailableFake) Envelope() Envelope        { return m.envelope }
func (m mailableFake) Content() (Content, error) { return m.content, m.err }
func (m mailableFake) Attachments() []Attachment { return m.attachments }

type transportFake struct {
	message Message
	err     error
}

func (t *transportFake) Send(_ context.Context, message Message) error {
	t.message = message
	return t.err
}

type dispatcherFake struct {
	job     SendJob
	options QueueOptions
	err     error
}

func (d *dispatcherFake) DispatchMessage(_ context.Context, job SendJob, options QueueOptions) (*QueuedMessageInfo, error) {
	d.job = job
	d.options = options
	return &QueuedMessageInfo{ID: "job-1", Queue: options.Queue}, d.err
}

func TestMailerSendRendersDefaultSenderAndAttachments(t *testing.T) {
	transport := &transportFake{}
	mailer := NewMailer(Address{Name: "App", Address: "hello@example.com"}, transport, nil)
	mailable := mailableFake{
		envelope: Envelope{
			To:      []Address{{Address: "user@example.com"}},
			Subject: "Subject",
		},
		content: Content{Text: "Plain", HTML: "<p>HTML</p>"},
		attachments: []Attachment{{
			Filename:    "hello.txt",
			ContentType: "text/plain",
			Data:        []byte("attachment"),
		}},
	}

	if err := mailer.Send(context.Background(), mailable); err != nil {
		t.Fatal(err)
	}
	if transport.message.Envelope.From.Address != "hello@example.com" {
		t.Fatalf("unexpected sender: %+v", transport.message.Envelope.From)
	}
	if transport.message.Content.HTML != "<p>HTML</p>" || string(transport.message.Attachments[0].Data) != "attachment" {
		t.Fatalf("unexpected rendered message: %+v", transport.message)
	}
}

func TestMailerQueueUsesMailDefaultsAndRenderedPayload(t *testing.T) {
	dispatcher := &dispatcherFake{}
	mailer := NewMailer(Address{Address: "hello@example.com"}, nil, dispatcher)
	info, err := mailer.Queue(context.Background(), mailableFake{
		envelope: Envelope{To: []Address{{Address: "user@example.com"}}, Subject: "Queued"},
		content:  Content{Text: "Hello"},
	}, QueueOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if info.Queue != "mail" {
		t.Fatalf("unexpected dispatched job: info=%+v", info)
	}
	if dispatcher.options.Queue != "mail" || dispatcher.options.MaxRetry != 3 || dispatcher.options.Timeout != 30*time.Second {
		t.Fatalf("unexpected queue options: %+v", dispatcher.options)
	}
	if dispatcher.job.Message.Content.Text != "Hello" {
		t.Fatalf("unexpected queued message: %+v", dispatcher.job.Message)
	}
}

func TestMailerReturnsRenderAndTransportErrors(t *testing.T) {
	renderErr := errors.New("render failed")
	mailer := NewMailer(Address{Address: "hello@example.com"}, &transportFake{}, nil)
	if err := mailer.Send(context.Background(), mailableFake{err: renderErr}); !errors.Is(err, renderErr) {
		t.Fatalf("got %v, want render error", err)
	}

	sendErr := errors.New("send failed")
	mailer = NewMailer(Address{Address: "hello@example.com"}, &transportFake{err: sendErr}, nil)
	err := mailer.Send(context.Background(), mailableFake{
		envelope: Envelope{To: []Address{{Address: "user@example.com"}}, Subject: "Subject"},
		content:  Content{Text: "Hello"},
	})
	if !errors.Is(err, sendErr) {
		t.Fatalf("got %v, want transport error", err)
	}
}

func TestWelcomeRendersTextAndHTML(t *testing.T) {
	content, err := (Welcome{Username: "Rifki", Email: "rifki@example.com"}).Content()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(content.Text, "Rifki") || !strings.Contains(content.HTML, "Rifki") {
		t.Fatalf("welcome content did not include username: %+v", content)
	}
}
