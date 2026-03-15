package router

import (
	"github.com/gofiber/fiber/v3"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/delivery/http/handler"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/usecase"
)

func Setup(app *fiber.App, userUsecase usecase.UserUsecase) {
	api := app.Group("/api")

	userHandler := handler.NewUserHandler(&userUsecase)

	api.Get("/users", userHandler.GetUsers)
	api.Get("/users/:id", userHandler.GetUserByID)
	api.Post("/users", userHandler.CreateUser)
	api.Put("/users/:id", userHandler.UpdateUser)
	api.Delete("/users/:id", userHandler.DeleteUser)
}
