package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/bootstrap"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/config"
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
	dispatcher, err := bootstrap.Dispatcher(cfg)
	if err != nil {
		log.Fatalf("Failed to build queue dispatcher: %v", err)
	}
	if closer, ok := dispatcher.(interface{ Close() error }); ok {
		defer closer.Close()
	}
	runner := scheduler.NewRunner(registry, dispatcher)

	log.Printf("Scheduler is running with timezone %s and queue driver %s", cfg.Scheduler.Timezone, cfg.Queue.Driver)
	for {
		now := time.Now()
		next := now.Truncate(time.Minute).Add(time.Minute)
		timer := time.NewTimer(time.Until(next))
		select {
		case <-ctx.Done():
			timer.Stop()
			log.Println("Stopping scheduler")
			return
		case tick := <-timer.C:
			if err := runner.Run(ctx, tick); err != nil {
				log.Printf("Scheduler tick failed: %v", err)
			}
		}
	}
}
