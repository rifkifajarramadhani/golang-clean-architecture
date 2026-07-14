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
)

func serveCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Run the HTTP API server",
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

			return runServer(ctx, cfg, logger, db, dispatcher)
		},
	}
}
