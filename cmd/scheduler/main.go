package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/adapter/logging"
	mysqladapter "github.com/rifkifajarramadhani/golang-clean-architecture/internal/adapter/mysql"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/bootstrap"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/config"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/scheduler"
	"gorm.io/gorm"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	logger, err := logging.New(cfg.Logging)
	if err != nil {
		log.Fatalf("create logger: %v", err)
	}
	defer func() { _ = logger.Close() }()
	var db = (*gorm.DB)(nil)
	if cfg.Queue.Driver == config.QueueDriverDatabase {
		db, err = mysqladapter.Open(ctx, cfg.Database.DSN, logger.Logger)
		if err != nil {
			logger.ErrorContext(ctx, "connect to database failed", "error", err)
			return
		}
		defer func() { _ = mysqladapter.Close(db) }()
	}
	registry, err := bootstrap.ScheduleRegistry(cfg)
	if err != nil {
		logger.ErrorContext(ctx, "build schedule registry failed", "error", err)
		return
	}
	dispatcher, err := bootstrap.Dispatcher(cfg, db)
	if err != nil {
		logger.ErrorContext(ctx, "build queue dispatcher failed", "error", err)
		return
	}
	if closer, ok := dispatcher.(interface{ Close() error }); ok {
		defer func() { _ = closer.Close() }()
	}
	runner := scheduler.NewRunner(registry, dispatcher)
	logger.Info("scheduler running", "timezone", cfg.Scheduler.Timezone, "queue_driver", cfg.Queue.Driver)
	for {
		now := time.Now()
		timer := time.NewTimer(time.Until(now.Truncate(time.Minute).Add(time.Minute)))
		select {
		case <-ctx.Done():
			timer.Stop()
			logger.Info("scheduler stopped")
			return
		case tick := <-timer.C:
			if err := runner.Run(ctx, tick); err != nil {
				logger.ErrorContext(ctx, "scheduler tick failed", "error", err)
			}
		}
	}
}
