package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/bootstrap"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/platform/config"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/platform/httpserver"
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
	db, err := bootstrap.Database(ctx, cfg)
	if err != nil {
		appLogger.Error("database startup failed", "error", err)
		os.Exit(1)
	}
	dispatcher := bootstrap.Dispatcher(cfg)
	defer func() { _ = dispatcher.(interface{ Close() error }).Close() }()
	redisClient := bootstrap.RedisClient(cfg)
	defer func() { _ = redisClient.Close() }()
	identityServices := bootstrap.IdentityService(cfg, db, dispatcher, appLogger)
	server := httpserver.New(cfg, appLogger, db, redisClient, identityServices.Service, identityServices.Tokens)

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := server.App.ShutdownWithContext(shutdownCtx); err != nil {
			appLogger.Error("server shutdown failed", "error", err)
		}
	}()
	appLogger.Info("server starting", "port", cfg.App.Port)
	if err := server.App.Listen(":" + cfg.App.Port); err != nil {
		appLogger.Error("server stopped", "error", err)
		os.Exit(1)
	}
}
