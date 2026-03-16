package main

import (
	"log"

	"github.com/gofiber/fiber/v3"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/config"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/delivery/http/router"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/infrastructure/database"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/infrastructure/logger"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/infrastructure/security"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/repository"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/usecase"
)

func main() {
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
	authUsecase := usecase.NewAuthUsecase(userRepo, jwtService)

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c fiber.Ctx, err error) error {
			logger.Logger.Println("Fiber error:", err)
			return c.Status(500).JSON(fiber.Map{"error": "Internal server error"})
		},
	})

	router.Setup(app, userUsecase, authUsecase, jwtService)

	port := cfg.App.Port
	log.Printf("Server is running on port %s", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
