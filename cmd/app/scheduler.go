package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"
	"time"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/adapter/database"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/adapter/logging"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/bootstrap"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/config"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/queue"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/scheduler"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

func schedulerCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "scheduler",
		Short: "Run the schedule ticker",
		RunE: func(_ *cobra.Command, _ []string) error {
			ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()

			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}

			logger, err := logging.New(cfg.Logging)
			if err != nil {
				return fmt.Errorf("create logger: %w", err)
			}

			defer func() { _ = logger.Close() }()

			db := (*gorm.DB)(nil)
			if cfg.Queue.Driver == config.QueueDriverDatabase {
				db, err = database.Open(ctx, cfg.Database, logger.Logger)
				if err != nil {
					logger.ErrorContext(ctx, "connect to database failed", "error", err)
					return fmt.Errorf("connect to database: %w", err)
				}

				defer func() { _ = database.Close(db) }()
			}

			dispatcher, err := bootstrap.Dispatcher(cfg, db)
			if err != nil {
				logger.ErrorContext(ctx, "build queue dispatcher failed", "error", err)
				return fmt.Errorf("build queue dispatcher: %w", err)
			}

			if closer, ok := dispatcher.(interface{ Close() error }); ok {
				defer func() { _ = closer.Close() }()
			}

			return runScheduler(ctx, cfg, logger, dispatcher)
		},
	}
}

func runScheduler(ctx context.Context, cfg *config.Config, logger *logging.Logger, dispatcher queue.Dispatcher) error {
	registry, err := bootstrap.ScheduleRegistry(cfg)
	if err != nil {
		logger.ErrorContext(ctx, "build schedule registry failed", "error", err)
		return fmt.Errorf("build schedule registry: %w", err)
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

			return nil
		case tick := <-timer.C:
			if err := runner.Run(ctx, tick); err != nil {
				logger.ErrorContext(ctx, "scheduler tick failed", "error", err)
			}
		}
	}
}
