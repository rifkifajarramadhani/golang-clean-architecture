package logger

import (
	"log/slog"
	"os"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/platform/config"
)

func New(cfg *config.Config) *slog.Logger {
	level := slog.LevelInfo
	if cfg.App.Environment == config.EnvironmentDevelopment {
		level = slog.LevelDebug
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})).
		With("service", cfg.App.Name, "environment", cfg.App.Environment)
}
