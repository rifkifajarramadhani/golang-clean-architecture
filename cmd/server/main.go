package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/gofiber/fiber/v3"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/bootstrap"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/config"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/delivery/http/router"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/infrastructure/database"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/infrastructure/logger"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/infrastructure/security"
	appmail "github.com/rifkifajarramadhani/golang-clean-architecture/internal/mail"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/repository"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/usecase"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	logger.Init()

	db, err := database.NewConnection(cfg.Database.DSN)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	jwtService := security.NewJWTService(
		cfg.Auth.JWTAccessSecret,
		cfg.Auth.JWTRefreshSecret,
		cfg.Auth.AccessTTLMinutes,
		cfg.Auth.RefreshTTLHours,
	)

	userRepo := repository.NewUserRepository(db)
	userUsecase := usecase.NewUserUsecase(userRepo)
	dispatcher, err := bootstrap.Dispatcher(cfg)
	if err != nil {
		log.Fatalf("Failed to build queue dispatcher: %v", err)
	}
	if closer, ok := dispatcher.(interface{ Close() error }); ok {
		defer closer.Close()
	}
	mailer := appmail.NewMailer(appmail.Address{
		Name:    cfg.Mail.FromName,
		Address: cfg.Mail.FromAddress,
	}, nil, dispatcher)
	authUsecase := usecase.NewAuthUsecase(userRepo, jwtService, mailer)

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			logger.Logger.Println("Fiber error:", err)
			return c.Status(500).JSON(fiber.Map{"error": "Internal server error"})
		},
	})

	router.Setup(app, userUsecase, authUsecase, jwtService)

	port := cfg.App.Port
	go func() {
		<-ctx.Done()
		if err := app.Shutdown(); err != nil {
			log.Printf("Failed to gracefully stop server: %v", err)
		}
	}()

	log.Printf("Server is running on port %s", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
