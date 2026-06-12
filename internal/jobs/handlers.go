package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	appmail "github.com/rifkifajarramadhani/golang-clean-architecture/internal/mail"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/queue"
)

type RefreshTokenCleaner interface {
	CleanupRefreshTokens(context.Context, time.Time) (int64, error)
}

func RegisterHandlers(registry *queue.HandlerRegistry, maintenance RefreshTokenCleaner, mailTransport appmail.Transport, logger *slog.Logger) error {
	if err := registry.Register(TypeDemoLog, func(_ context.Context, payload json.RawMessage) error {
		var job DemoLog
		if err := json.Unmarshal(payload, &job); err != nil {
			return fmt.Errorf("decode demo log job: %w", err)
		}
		logger.Info("demo job", "message", job.Message)
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
		logger.Info("refresh-token cleanup completed", "deleted", deleted)
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
