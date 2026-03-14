package handler

import (
	"github.com/gofiber/fiber/v3"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/domain"
)

type UserHandler struct {
	user domain.User
}

func NewUserHandler() *UserHandler {
	return &UserHandler{}
}

func (h *UserHandler) GetUsers(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "Get User",
	})
}

func (h *UserHandler) GetUserByID(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "Get User By ID",
	})
}

func (h *UserHandler) CreateUser(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "Create User",
	})
}

func (h *UserHandler) UpdateUser(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "Update User",
	})
}

func (h *UserHandler) DeleteUser(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"message": "Delete User",
	})
}
