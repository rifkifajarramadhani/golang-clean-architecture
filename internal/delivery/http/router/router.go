package router

import (
	"github.com/gofiber/fiber/v3"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/delivery/http/handler"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/delivery/http/middleware"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/infrastructure/security"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/usecase"
)

func Setup(app *fiber.App, userUsecase *usecase.UserUsecase, authUsecase *usecase.AuthUsecase, jwtService *security.JWTService) {
	api := app.Group("/api")
	auth := api.Group("/auth")

	authHandler := handler.NewAuthHandler(authUsecase)
	auth.Post("/register", authHandler.Register)
	auth.Post("/login", authHandler.Login)
	auth.Post("/refresh", authHandler.Refresh)

	protected := api.Group("", middleware.JWTAuth(jwtService))
	protected.Get("/auth/me", authHandler.Me)

	userHandler := handler.NewUserHandler(userUsecase)
	protected.Get("/users", userHandler.GetUsers)
	protected.Get("/users/:id", userHandler.GetUserByID)
	protected.Post("/users", userHandler.CreateUser)
	protected.Put("/users/:id", userHandler.UpdateUser)
	protected.Delete("/users/:id", userHandler.DeleteUser)
}
