package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/helmet"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/adapter/http/router"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/adapter/logging"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/bootstrap"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/config"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/queue"
	"gorm.io/gorm"
)

func runServer(ctx context.Context, cfg *config.Config, logger *logging.Logger, db *gorm.DB, dispatcher queue.Dispatcher) error {
	services := bootstrap.WireHTTPServices(cfg, db, logger.Logger, dispatcher)
	app := fiber.New(fiber.Config{
		BodyLimit: 64 * 1024, ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second,
		ErrorHandler: func(c fiber.Ctx, err error) error {
			var fiberErr *fiber.Error
			if errors.As(err, &fiberErr) {
				return c.Status(fiberErr.Code).JSON(fiber.Map{"error": fiberErr.Message})
			}

			logger.ErrorContext(c.Context(), "fiber error", "error", err)

			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
		},
	})
	app.Use(helmet.New())
	router.Setup(app, services.Users, services.Auth, services.Tokens, logger.Logger)
	app.Get("/health/live", func(c fiber.Ctx) error { return c.JSON(fiber.Map{"status": "ok"}) })
	app.Get("/health/ready", func(c fiber.Ctx) error {
		sqlDB, dbErr := db.DB()
		if dbErr != nil || sqlDB.PingContext(c.Context()) != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"status": "unavailable"})
		}

		return c.JSON(fiber.Map{"status": "ok"})
	})

	go func() {
		<-ctx.Done()

		done := make(chan error, 1)
		go func() { done <- app.Shutdown() }()

		select {
		case shutdownErr := <-done:
			if shutdownErr != nil {
				logger.Error("graceful server shutdown failed", "error", shutdownErr)
			}
		case <-time.After(15 * time.Second):
			logger.Error("graceful server shutdown timed out")
		}
	}()

	logger.Info("server running", "port", cfg.App.Port)

	if err := app.Listen(":" + cfg.App.Port); err != nil && ctx.Err() == nil {
		return fmt.Errorf("listen: %w", err)
	}

	return nil
}
