package main

import (
	"context"
	"fmt"
	"os/signal"
	"syscall"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/adapter/database"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/adapter/logging"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/bootstrap"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/config"
	"github.com/spf13/cobra"
	"gorm.io/gorm"
)

func workerCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "worker",
		Short: "Run the queue worker",
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

			db, err := database.Open(ctx, cfg.Database, logger.Logger)
			if err != nil {
				logger.ErrorContext(ctx, "connect to database failed", "error", err)
				return fmt.Errorf("connect to database: %w", err)
			}

			defer func() { _ = database.Close(db) }()

			return runWorker(ctx, cfg, logger, db)
		},
	}
}

func runWorker(ctx context.Context, cfg *config.Config, logger *logging.Logger, db *gorm.DB) error {
	worker, err := bootstrap.Worker(cfg, db, logger.Logger)
	if err != nil {
		logger.ErrorContext(ctx, "build worker failed", "error", err)
		return fmt.Errorf("build worker: %w", err)
	}

	logger.Info("worker running", "driver", cfg.Queue.Driver, "queues", cfg.Queue.Queues)

	if err := worker.Run(ctx); err != nil {
		return fmt.Errorf("run worker: %w", err)
	}

	logger.Info("worker stopped")

	return nil
}
