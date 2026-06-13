package main

import (
	"context"
	"errors"
	"log"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/adapter/http/router"
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
	appLogger, err := logging.New(cfg.Logging)
	if err != nil {
		log.Fatalf("create logger: %v", err)
	}
	defer func() { _ = appLogger.Close() }()
	db, err := mysqladapter.Open(ctx, cfg.Database.DSN, appLogger.Logger)
	if err != nil {
		appLogger.ErrorContext(ctx, "connect to database failed", "error", err)
		return
	}
	defer func() { _ = mysqladapter.Close(db) }()

	dispatcher, err := bootstrap.Dispatcher(cfg, db)
	if err != nil {
		appLogger.ErrorContext(ctx, "build queue dispatcher failed", "error", err)
		return
	}
	if closer, ok := dispatcher.(interface{ Close() error }); ok {
		defer func() { _ = closer.Close() }()
	}
	services := bootstrap.WireHTTPServices(cfg, db, appLogger.Logger, dispatcher)

	app := fiber.New(fiber.Config{
		BodyLimit: 64 * 1024, ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second,
		ErrorHandler: func(c fiber.Ctx, err error) error {
			var fiberErr *fiber.Error
			if errors.As(err, &fiberErr) {
				return c.Status(fiberErr.Code).JSON(fiber.Map{"error": fiberErr.Message})
			}
			appLogger.ErrorContext(c.Context(), "fiber error", "error", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
		},
	})
	app.Use(helmet.New())
	router.Setup(app, services.Users, services.Auth, services.Tokens, appLogger.Logger)
	go func() {
		<-ctx.Done()
		if err := app.Shutdown(); err != nil {
			appLogger.Error("graceful server shutdown failed", "error", err)
		}
	}()
	appLogger.Info("server running", "port", cfg.App.Port)
	if err := app.Listen(":" + cfg.App.Port); err != nil {
		appLogger.Error("server stopped", "error", err)
	}
}
