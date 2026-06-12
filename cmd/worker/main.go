package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/bootstrap"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/config"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	worker, err := bootstrap.Worker(cfg)
	if err != nil {
		log.Fatalf("Failed to build worker: %v", err)
	}
	log.Printf("Worker is running with driver %s and queues: %v", cfg.Queue.Driver, cfg.Queue.Queues)
	if err := worker.Run(ctx); err != nil {
		log.Fatalf("Failed to start worker: %v", err)
	}
	log.Println("Stopping worker")
}
