package jobs

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	appmail "github.com/rifkifajarramadhani/golang-clean-architecture/internal/mail"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/queue"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/usecase"
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
	if err := RegisterHandlers(registry, usecase.NewMaintenanceUsecase(maintenanceRepositoryFake{}), &mailTransportFake{}); err != nil {
		t.Fatal(err)
	}
	handler := registry.Handlers()[TypeDemoLog]
	if err := handler(context.Background(), json.RawMessage("{")); err == nil {
		t.Fatal("expected malformed payload error")
	}
}

func TestMailHandlerRejectsMalformedPayload(t *testing.T) {
	registry := queue.NewHandlerRegistry()
	if err := RegisterHandlers(registry, usecase.NewMaintenanceUsecase(maintenanceRepositoryFake{}), &mailTransportFake{}); err != nil {
		t.Fatal(err)
	}
	if err := registry.Handlers()[appmail.TypeSend](context.Background(), json.RawMessage("{")); err == nil {
		t.Fatal("expected malformed payload error")
	}
}

func TestMailHandlerForwardsRenderedMessage(t *testing.T) {
	transport := &mailTransportFake{}
	registry := queue.NewHandlerRegistry()
	if err := RegisterHandlers(registry, usecase.NewMaintenanceUsecase(maintenanceRepositoryFake{}), transport); err != nil {
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
