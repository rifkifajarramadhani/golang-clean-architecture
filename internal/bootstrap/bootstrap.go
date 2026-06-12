package bootstrap

import (
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/config"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/infrastructure/database"
	mailinfra "github.com/rifkifajarramadhani/golang-clean-architecture/internal/infrastructure/mail"
	queueinfra "github.com/rifkifajarramadhani/golang-clean-architecture/internal/infrastructure/queue"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/jobs"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/queue"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/repository"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/scheduler"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/usecase"
)

func RedisOptions(cfg *config.Config) asynq.RedisClientOpt {
	return queueinfra.NewRedisOptions(cfg.Redis)
}

func Dispatcher(cfg *config.Config) (queue.Dispatcher, error) {
	switch cfg.Queue.Driver {
	case config.QueueDriverRedis:
		return queueinfra.NewDispatcher(RedisOptions(cfg)), nil
	case config.QueueDriverDatabase:
		db, err := database.NewConnection(cfg.Database.DSN)
		if err != nil {
			return nil, err
		}
		return queueinfra.NewDatabaseDispatcher(db), nil
	default:
		return nil, fmt.Errorf("unsupported queue driver %q", cfg.Queue.Driver)
	}
}

func Inspector(cfg *config.Config) (queue.Inspector, error) {
	switch cfg.Queue.Driver {
	case config.QueueDriverRedis:
		return queueinfra.NewRedisInspector(RedisOptions(cfg)), nil
	case config.QueueDriverDatabase:
		db, err := database.NewConnection(cfg.Database.DSN)
		if err != nil {
			return nil, err
		}
		return queueinfra.NewDatabaseInspector(db, cfg.Queue.Queues), nil
	default:
		return nil, fmt.Errorf("unsupported queue driver %q", cfg.Queue.Driver)
	}
}

func Worker(cfg *config.Config) (queue.Worker, error) {
	db, err := database.NewConnection(cfg.Database.DSN)
	if err != nil {
		return nil, err
	}
	userRepo := repository.NewUserRepository(db)
	maintenance := usecase.NewMaintenanceUsecase(userRepo)
	mailTransport := mailinfra.NewSMTPTransport(cfg.Mail)
	registry := queue.NewHandlerRegistry()
	if err := jobs.RegisterHandlers(registry, maintenance, mailTransport); err != nil {
		return nil, fmt.Errorf("register job handlers: %w", err)
	}
	switch cfg.Queue.Driver {
	case config.QueueDriverRedis:
		return queueinfra.NewRedisWorker(RedisOptions(cfg), cfg.Queue, registry), nil
	case config.QueueDriverDatabase:
		return queueinfra.NewDatabaseWorker(db, cfg.Queue, registry), nil
	default:
		return nil, fmt.Errorf("unsupported queue driver %q", cfg.Queue.Driver)
	}
}

func ScheduleRegistry(cfg *config.Config) (*scheduler.Registry, error) {
	registry, err := scheduler.NewRegistry(cfg.Scheduler.Timezone)
	if err != nil {
		return nil, err
	}
	err = registry.Register(scheduler.Definition{
		Name:     "cleanup-refresh-tokens",
		Cron:     "0 0 * * *",
		Timezone: cfg.Scheduler.Timezone,
		Job: func() queue.Job {
			return jobs.CleanupRefreshTokens{}
		},
		DispatchOptions: queue.DispatchOptions{
			Queue:     "maintenance",
			MaxRetry:  3,
			Retention: 2 * time.Minute,
		},
	})
	return registry, err
}
