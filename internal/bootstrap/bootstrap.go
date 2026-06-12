package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/identity"
	identitymail "github.com/rifkifajarramadhani/golang-clean-architecture/internal/identity/mail"
	identitymysql "github.com/rifkifajarramadhani/golang-clean-architecture/internal/identity/mysql"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/jobs"
	appmail "github.com/rifkifajarramadhani/golang-clean-architecture/internal/mail"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/platform/config"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/platform/database"
	mailinfra "github.com/rifkifajarramadhani/golang-clean-architecture/internal/platform/mail"
	queueinfra "github.com/rifkifajarramadhani/golang-clean-architecture/internal/platform/queue"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/platform/security"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/queue"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/scheduler"
	"gorm.io/gorm"
)

type Identity struct {
	Service *identity.Service
	Tokens  *security.JWTService
}

func RedisOptions(cfg *config.Config) asynq.RedisClientOpt {
	return queueinfra.NewRedisOptions(cfg.Redis)
}

func RedisClient(cfg *config.Config) *redis.Client {
	return redis.NewClient(&redis.Options{Addr: cfg.Redis.Address, Password: cfg.Redis.Password, DB: cfg.Redis.DB})
}

func Database(ctx context.Context, cfg *config.Config) (*gorm.DB, error) {
	return database.NewConnection(ctx, cfg.Database)
}

func Dispatcher(cfg *config.Config) queue.Dispatcher {
	return queueinfra.NewDispatcher(RedisOptions(cfg))
}

func Inspector(cfg *config.Config) queue.Inspector {
	return queueinfra.NewRedisInspector(RedisOptions(cfg))
}

func IdentityService(cfg *config.Config, db *gorm.DB, dispatcher queue.Dispatcher, logger *slog.Logger) Identity {
	tokens := security.NewJWTService(
		cfg.Auth.JWTAccessSecret, cfg.Auth.JWTRefreshSecret,
		cfg.Auth.AccessTTL(), cfg.Auth.RefreshTTL(), cfg.Auth.Issuer, cfg.Auth.Audience,
	)
	repo := identitymysql.NewRepository(db)
	mailer := appmail.NewMailer(appmail.Address{Name: cfg.Mail.FromName, Address: cfg.Mail.FromAddress}, nil, dispatcher)
	return Identity{Service: identity.NewService(repo, tokens, identitymail.NewDispatcher(mailer), logger), Tokens: tokens}
}

func Worker(ctx context.Context, cfg *config.Config, logger *slog.Logger) (queue.Worker, error) {
	db, err := Database(ctx, cfg)
	if err != nil {
		return nil, err
	}
	dispatcher := Dispatcher(cfg)
	services := IdentityService(cfg, db, dispatcher, logger)
	mailTransport := mailinfra.NewSMTPTransport(cfg.Mail)
	registry := queue.NewHandlerRegistry()
	if err := jobs.RegisterHandlers(registry, services.Service, mailTransport, logger); err != nil {
		return nil, fmt.Errorf("register job handlers: %w", err)
	}
	return queueinfra.NewRedisWorker(RedisOptions(cfg), cfg.Queue, registry), nil
}

func ScheduleRegistry(cfg *config.Config) (*scheduler.Registry, error) {
	registry, err := scheduler.NewRegistry(cfg.Scheduler.Timezone)
	if err != nil {
		return nil, err
	}
	err = registry.Register(scheduler.Definition{
		Name: "cleanup-refresh-tokens", Cron: "0 0 * * *", Timezone: cfg.Scheduler.Timezone,
		Job:             func() queue.Job { return jobs.CleanupRefreshTokens{} },
		DispatchOptions: queue.DispatchOptions{Queue: "maintenance", MaxRetry: 3, Retention: 2 * time.Minute},
	})
	return registry, err
}
