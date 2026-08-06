package main

import (
	"context"
	"fmt"
	"os/signal"
	"sync"
	"syscall"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/adapter/database"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/adapter/logging"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/bootstrap"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/config"
	"github.com/spf13/cobra"
)

func allCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "all",
		Short: "Run the server, worker, and scheduler in a single process (shared config, DB pool, and dispatcher)",
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

			dispatcher, err := bootstrap.Dispatcher(cfg, db)
			if err != nil {
				logger.ErrorContext(ctx, "build queue dispatcher failed", "error", err)
				return fmt.Errorf("build queue dispatcher: %w", err)
			}

			if closer, ok := dispatcher.(interface{ Close() error }); ok {
				defer func() { _ = closer.Close() }()
			}

			logger.Info("running all components in a single process")

			ctx, cancel := context.WithCancel(ctx)
			defer cancel()

			var (
				wg       sync.WaitGroup
				mu       sync.Mutex
				firstErr error
			)

			run := func(name string, fn func() error) {
				defer wg.Done()
				defer cancel()

				if err := fn(); err != nil {
					logger.ErrorContext(ctx, "component failed", "component", name, "error", err)

					mu.Lock()
					if firstErr == nil {
						firstErr = fmt.Errorf("%s: %w", name, err)
					}
					mu.Unlock()
				}
			}

			wg.Add(3)

			go run("server", func() error { return runServer(ctx, cfg, logger, db, dispatcher) })
			go run("worker", func() error { return runWorker(ctx, cfg, logger, db) })
			go run("scheduler", func() error { return runScheduler(ctx, cfg, logger, dispatcher) })

			wg.Wait()

			return firstErr
		},
	}
}
