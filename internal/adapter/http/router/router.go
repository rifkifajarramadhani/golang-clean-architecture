package router

import (
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/limiter"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/adapter/http/handler"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/adapter/http/middleware"
)

func Setup(app *fiber.App, users handler.UserService, auth handler.AuthService, tokens middleware.AccessTokenValidator, logger *slog.Logger) {
	api := app.Group("/api")
	authGroup := api.Group("/auth", limiter.New(limiter.Config{
		Max: 20, Expiration: time.Minute,
		LimitReached: func(c fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "too many requests"})
		},
	}))
	authHandler := handler.NewAuthHandler(auth, logger)
	authGroup.Post("/register", authHandler.Register)
	authGroup.Post("/login", authHandler.Login)
	authGroup.Post("/refresh", authHandler.Refresh)
	authGroup.Post("/verify-email", authHandler.VerifyEmail)
	authGroup.Post("/resend-verification", authHandler.ResendVerification)

	protected := api.Group("", middleware.JWTAuth(tokens, users))
	protected.Get("/auth/me", authHandler.Me)

	userHandler := handler.NewUserHandler(users, auth, logger)
	protected.Patch("/users/me", userHandler.UpdateSelf)
	protected.Put("/users/me/password", userHandler.ChangePassword)
	protected.Delete("/users/me", userHandler.DeleteSelf)

	admin := protected.Group("/users", middleware.AdminOnly)
	admin.Get("/", userHandler.GetUsers)
	admin.Post("/", userHandler.CreateUser)
	admin.Get("/:id", userHandler.GetUserByID)
	admin.Put("/:id", userHandler.UpdateUser)
	admin.Put("/:id/role", userHandler.ChangeRole)
	admin.Delete("/:id", userHandler.DeleteUser)
}
