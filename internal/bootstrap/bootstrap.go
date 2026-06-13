package bootstrap

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/hibiken/asynq"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/adapter/cron"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/adapter/jobs"
	queueinfra "github.com/rifkifajarramadhani/golang-clean-architecture/internal/adapter/queue"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/adapter/smtp"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/auth"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/config"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/queue"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/scheduler"
	"gorm.io/gorm"
)

func RedisOptions(cfg *config.Config) asynq.RedisClientOpt {
	return queueinfra.NewRedisOptions(cfg.Redis)
}

func Dispatcher(cfg *config.Config, db *gorm.DB) (queue.Dispatcher, error) {
	switch cfg.Queue.Driver {
	case config.QueueDriverRedis:
		return queueinfra.NewDispatcher(RedisOptions(cfg)), nil
	case config.QueueDriverDatabase:
		if db == nil {
			return nil, fmt.Errorf("database connection is required for database queue driver")
		}
		return queueinfra.NewDatabaseDispatcher(db), nil
	default:
		return nil, fmt.Errorf("unsupported queue driver %q", cfg.Queue.Driver)
	}
}

func Inspector(cfg *config.Config, db *gorm.DB) (queue.Inspector, error) {
	switch cfg.Queue.Driver {
	case config.QueueDriverRedis:
		return queueinfra.NewRedisInspector(RedisOptions(cfg)), nil
	case config.QueueDriverDatabase:
		if db == nil {
			return nil, fmt.Errorf("database connection is required for database queue driver")
		}
		return queueinfra.NewDatabaseInspector(db, cfg.Queue.Queues), nil
	default:
		return nil, fmt.Errorf("unsupported queue driver %q", cfg.Queue.Driver)
	}
}

func Worker(cfg *config.Config, db *gorm.DB, logger *slog.Logger) (queue.Worker, error) {
	if db == nil {
		return nil, fmt.Errorf("database connection is required by worker job handlers")
	}
	repository := mysqlRepository(db)
	maintenance := auth.NewMaintenanceService(repository)
	mailTransport, err := smtp.NewTransport(cfg.Mail)
	if err != nil {
		return nil, err
	}
	registry := queue.NewHandlerRegistry()
	if err := jobs.RegisterHandlers(registry, maintenance, mailTransport, logger); err != nil {
		return nil, fmt.Errorf("register job handlers: %w", err)
	}
	switch cfg.Queue.Driver {
	case config.QueueDriverRedis:
		return queueinfra.NewRedisWorker(RedisOptions(cfg), cfg.Queue, registry), nil
	case config.QueueDriverDatabase:
		return queueinfra.NewDatabaseWorker(db, cfg.Queue, registry, logger), nil
	default:
		return nil, fmt.Errorf("unsupported queue driver %q", cfg.Queue.Driver)
	}
}

func ScheduleRegistry(cfg *config.Config) (*scheduler.Registry, error) {
	registry, err := scheduler.NewRegistry(cfg.Scheduler.Timezone, cron.Parser{})
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
