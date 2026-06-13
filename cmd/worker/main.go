package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/adapter/logging"
	mysqladapter "github.com/rifkifajarramadhani/golang-clean-architecture/internal/adapter/mysql"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/bootstrap"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/config"
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
	db, err := mysqladapter.Open(ctx, cfg.Database.DSN, logger.Logger)
	if err != nil {
		logger.ErrorContext(ctx, "connect to database failed", "error", err)
		return
	}
	defer func() { _ = mysqladapter.Close(db) }()
	worker, err := bootstrap.Worker(cfg, db, logger.Logger)
	if err != nil {
		logger.ErrorContext(ctx, "build worker failed", "error", err)
		return
	}
	logger.Info("worker running", "driver", cfg.Queue.Driver, "queues", cfg.Queue.Queues)
	if err := worker.Run(ctx); err != nil {
		logger.ErrorContext(ctx, "worker stopped with error", "error", err)
	}
	logger.Info("worker stopped")
}
