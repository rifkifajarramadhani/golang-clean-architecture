package jobs

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	appmail "github.com/rifkifajarramadhani/golang-clean-architecture/internal/mail"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/user"
)

type failingMailDispatcher struct{}

func (failingMailDispatcher) DispatchMessage(context.Context, appmail.SendJob, appmail.QueueOptions) (*appmail.QueuedMessageInfo, error) {
	return nil, errors.New("queue unavailable")
}

func TestWelcomeNotifierIsBestEffort(t *testing.T) {
	mailer := appmail.NewMailer(
		appmail.Address{Address: "hello@example.com"},
		nil,
		failingMailDispatcher{},
	)
	notifier := NewWelcomeNotifier(mailer, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err := notifier.NotifyWelcome(context.Background(), user.User{
		ID: 42, Username: "rifki", Email: "rifki@example.com",
	}); err == nil {
		t.Fatal("expected queue error")
	}
}
