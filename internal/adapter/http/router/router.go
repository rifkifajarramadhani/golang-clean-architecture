package router

import (
	"log/slog"

	"github.com/gofiber/fiber/v3"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/adapter/http/handler"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/adapter/http/middleware"
)

func Setup(app *fiber.App, users handler.UserService, auth handler.AuthService, tokens middleware.AccessTokenValidator, logger *slog.Logger) {
	api := app.Group("/api")
	authGroup := api.Group("/auth")
	authHandler := handler.NewAuthHandler(auth, logger)
	authGroup.Post("/register", authHandler.Register)
	authGroup.Post("/login", authHandler.Login)
	authGroup.Post("/refresh", authHandler.Refresh)

	protected := api.Group("", middleware.JWTAuth(tokens))
	protected.Get("/auth/me", authHandler.Me)
	userHandler := handler.NewUserHandler(users, logger)
	protected.Get("/users", userHandler.GetUsers)
	protected.Get("/users/:id", userHandler.GetUserByID)
	protected.Post("/users", userHandler.CreateUser)
	protected.Put("/users/:id", userHandler.UpdateUser)
	protected.Delete("/users/:id", userHandler.DeleteUser)
}
