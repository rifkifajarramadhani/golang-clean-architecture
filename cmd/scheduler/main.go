package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/bootstrap"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/platform/config"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/platform/logger"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/scheduler"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	registry, err := bootstrap.ScheduleRegistry(cfg)
	if err != nil {
		log.Fatalf("Failed to build schedule registry: %v", err)
	}
	appLogger := logger.New(cfg)
	dispatcher := bootstrap.Dispatcher(cfg)
	if closer, ok := dispatcher.(interface{ Close() error }); ok {
		defer func() { _ = closer.Close() }()
	}
	runner := scheduler.NewRunner(registry, dispatcher)

	appLogger.Info("scheduler starting", "timezone", cfg.Scheduler.Timezone)
	for {
		now := time.Now()
		next := now.Truncate(time.Minute).Add(time.Minute)
		timer := time.NewTimer(time.Until(next))
		select {
		case <-ctx.Done():
			timer.Stop()
			appLogger.Info("scheduler stopping")
			return
		case tick := <-timer.C:
			if err := runner.Run(ctx, tick); err != nil {
				appLogger.Error("scheduler tick failed", "error", err)
			}
		}
	}
}
