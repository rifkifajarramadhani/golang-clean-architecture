package main

import (
	"context"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/adapter/logging"
	mysqladapter "github.com/rifkifajarramadhani/golang-clean-architecture/internal/adapter/mysql"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/config"
	"gorm.io/gorm"
)

// openCLIDatabase opens the GORM pool only when the database queue driver is
// selected, returning (nil, nil) for the redis driver. Short-lived CLI
// subcommands build a logging.New logger for connection diagnostics — matching
// the long-running daemons instead of falling back to slog.Default().
func openCLIDatabase(ctx context.Context, cfg *config.Config) (*gorm.DB, error) {
	if cfg.Queue.Driver != config.QueueDriverDatabase {
		return nil, nil
	}
	logger, err := logging.New(cfg.Logging)
	if err != nil {
		return nil, err
	}
	defer func() { _ = logger.Close() }()
	return mysqladapter.Open(ctx, cfg.Database.DSN, logger.Logger)
}
