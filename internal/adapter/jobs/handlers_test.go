package jobs

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
	"time"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/auth"
	appmail "github.com/rifkifajarramadhani/golang-clean-architecture/internal/mail"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/queue"
)

type maintenanceRepositoryFake struct{}

func (maintenanceRepositoryFake) DeleteExpiredOrRevokedRefreshTokens(context.Context, time.Time) (int64, error) {
	return 0, nil
}

type mailTransportFake struct {
	message appmail.Message
}

func (f *mailTransportFake) Send(_ context.Context, message appmail.Message) error {
	f.message = message
	return nil
}

func TestDemoHandlerRejectsMalformedPayload(t *testing.T) {
	registry := queue.NewHandlerRegistry()
	if err := RegisterHandlers(registry, auth.NewMaintenanceService(maintenanceRepositoryFake{}), &mailTransportFake{}, slog.Default()); err != nil {
		t.Fatal(err)
	}
	handler := registry.Handlers()[TypeDemoLog]
	if err := handler(context.Background(), json.RawMessage("{")); err == nil {
		t.Fatal("expected malformed payload error")
	}
}

func TestMailHandlerRejectsMalformedPayload(t *testing.T) {
	registry := queue.NewHandlerRegistry()
	if err := RegisterHandlers(registry, auth.NewMaintenanceService(maintenanceRepositoryFake{}), &mailTransportFake{}, slog.Default()); err != nil {
		t.Fatal(err)
	}
	if err := registry.Handlers()[appmail.TypeSend](context.Background(), json.RawMessage("{")); err == nil {
		t.Fatal("expected malformed payload error")
	}
}

func TestMailHandlerForwardsRenderedMessage(t *testing.T) {
	transport := &mailTransportFake{}
	registry := queue.NewHandlerRegistry()
	if err := RegisterHandlers(registry, auth.NewMaintenanceService(maintenanceRepositoryFake{}), transport, slog.Default()); err != nil {
		t.Fatal(err)
	}
	payload, err := json.Marshal(appmail.SendJob{Message: appmail.Message{
		Envelope: appmail.Envelope{Subject: "Welcome"},
		Content:  appmail.Content{Text: "Hello"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Handlers()[appmail.TypeSend](context.Background(), payload); err != nil {
		t.Fatal(err)
	}
	if transport.message.Envelope.Subject != "Welcome" || transport.message.Content.Text != "Hello" {
		t.Fatalf("unexpected message: %+v", transport.message)
	}
}
