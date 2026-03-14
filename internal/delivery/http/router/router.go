package router

import (
	"github.com/gofiber/fiber/v3"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/delivery/http/handler"
)

func Setup(app *fiber.App) {
	api := app.Group("/api")

	userHandler := handler.NewUserHandler()

	api.Get("/users", userHandler.GetUsers)
	api.Get("/users/:id", userHandler.GetUserByID)
	api.Post("/users", userHandler.CreateUser)
	api.Put("/users/:id", userHandler.UpdateUser)
	api.Delete("/users/:id", userHandler.DeleteUser)
}
