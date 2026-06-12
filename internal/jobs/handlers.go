package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	appmail "github.com/rifkifajarramadhani/golang-clean-architecture/internal/mail"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/queue"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/usecase"
)

func RegisterHandlers(registry *queue.HandlerRegistry, maintenance *usecase.MaintenanceUsecase, mailTransport appmail.Transport) error {
	if err := registry.Register(TypeDemoLog, func(_ context.Context, payload json.RawMessage) error {
		var job DemoLog
		if err := json.Unmarshal(payload, &job); err != nil {
			return fmt.Errorf("decode demo log job: %w", err)
		}
		log.Printf("Demo job: %s", job.Message)
		return nil
	}); err != nil {
		return err
	}

	if err := registry.Register(TypeCleanupRefreshToken, func(ctx context.Context, payload json.RawMessage) error {
		var job CleanupRefreshTokens
		if err := json.Unmarshal(payload, &job); err != nil {
			return fmt.Errorf("decode cleanup refresh tokens job: %w", err)
		}
		deleted, err := maintenance.CleanupRefreshTokens(ctx, time.Now())
		if err != nil {
			return err
		}
		log.Printf("Deleted %d expired or revoked refresh tokens", deleted)
		return nil
	}); err != nil {
		return err
	}

	return registry.Register(appmail.TypeSend, func(ctx context.Context, payload json.RawMessage) error {
		var job appmail.SendJob
		if err := json.Unmarshal(payload, &job); err != nil {
			return fmt.Errorf("decode send mail job: %w", err)
		}
		if mailTransport == nil {
			return fmt.Errorf("mail transport is not configured")
		}
		return mailTransport.Send(ctx, job.Message)
	})
}
