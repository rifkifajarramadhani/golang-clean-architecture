package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v3"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/adapter/http/router"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/adapter/jobs"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/adapter/jwt"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/adapter/logging"
	mysqladapter "github.com/rifkifajarramadhani/golang-clean-architecture/internal/adapter/mysql"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/adapter/password"
	queueadapter "github.com/rifkifajarramadhani/golang-clean-architecture/internal/adapter/queue"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/auth"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/bootstrap"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/config"
	appmail "github.com/rifkifajarramadhani/golang-clean-architecture/internal/mail"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/user"
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

	repository := mysqladapter.NewUserRepository(db)
	hasher := password.Bcrypt{}
	users := user.NewService(repository, hasher)
	tokens := jwt.NewService(cfg.Auth.JWTAccessSecret, cfg.Auth.JWTRefreshSecret, cfg.Auth.AccessTTLMinutes, cfg.Auth.RefreshTTLHours)
	dispatcher, err := bootstrap.Dispatcher(cfg, db)
	if err != nil {
		appLogger.ErrorContext(ctx, "build queue dispatcher failed", "error", err)
		return
	}
	if closer, ok := dispatcher.(interface{ Close() error }); ok {
		defer func() { _ = closer.Close() }()
	}
	mailer := appmail.NewMailer(appmail.Address{Name: cfg.Mail.FromName, Address: cfg.Mail.FromAddress}, nil, queueadapter.NewMailDispatcher(dispatcher))
	authService := auth.NewService(repository, tokens, hasher, jobs.NewWelcomeNotifier(mailer, appLogger.Logger))

	app := fiber.New(fiber.Config{ErrorHandler: func(c fiber.Ctx, err error) error {
		appLogger.ErrorContext(c.Context(), "fiber error", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Internal server error"})
	}})
	router.Setup(app, users, authService, tokens, appLogger.Logger)
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
