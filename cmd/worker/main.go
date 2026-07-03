package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/adapter/logging"
	mysqladapter "github.com/rifkifajarramadhani/golang-clean-architecture/internal/adapter/mysql"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/bootstrap"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/config"
)

func main() {
	if err := run(); err != nil {
		log.Printf("worker failed: %v", err)
		os.Exit(1)
	}
}

func run() error {
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
	db, err := mysqladapter.Open(ctx, cfg.Database.DSN, logger.Logger)
	if err != nil {
		logger.ErrorContext(ctx, "connect to database failed", "error", err)
		return fmt.Errorf("connect to database: %w", err)
	}
	defer func() { _ = mysqladapter.Close(db) }()
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
