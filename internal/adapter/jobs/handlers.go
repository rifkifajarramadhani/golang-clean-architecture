package jobs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/auth"
	appmail "github.com/rifkifajarramadhani/golang-clean-architecture/internal/mail"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/queue"
)

func RegisterHandlers(registry *queue.HandlerRegistry, maintenance *auth.MaintenanceService, mailTransport appmail.Transport, logger *slog.Logger) error {
	if err := registry.Register(TypeDemoLog, func(ctx context.Context, payload json.RawMessage) error {
		var job DemoLog
		if err := json.Unmarshal(payload, &job); err != nil {
			return fmt.Errorf("decode demo log job: %w", err)
		}
		logger.InfoContext(ctx, "demo job", "message", job.Message)
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
		logger.InfoContext(ctx, "deleted expired or revoked refresh tokens", "count", deleted)
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
