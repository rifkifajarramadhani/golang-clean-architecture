package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/bootstrap"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/platform/config"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/platform/logger"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}
	appLogger := logger.New(cfg)
	worker, err := bootstrap.Worker(ctx, cfg, appLogger)
	if err != nil {
		appLogger.Error("build worker", "error", err)
		os.Exit(1)
	}
	appLogger.Info("worker starting", "queues", cfg.Queue.Queues)
	if err := worker.Run(ctx); err != nil {
		appLogger.Error("worker stopped", "error", err)
		os.Exit(1)
	}
}
