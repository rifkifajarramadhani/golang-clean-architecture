package handler

import (
	"github.com/gofiber/fiber/v3"
	dto "github.com/rifkifajarramadhani/golang-clean-architecture/internal/delivery/http/dto/user"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/domain"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/usecase"
)

type UserHandler struct {
	userUsecase usecase.UserUsecase
}

func NewUserHandler(userUsecase *usecase.UserUsecase) *UserHandler {
	return &UserHandler{
		userUsecase: *userUsecase,
	}
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
	var req dto.RegisterUserRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
		})
	}

	user := domain.User{
		Username: req.Username,
		Email:    req.Email,
	}

	err := h.userUsecase.CreateUser(&user)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to create user",
		})
	}

	res := dto.RegisterUserResponse{
		Username: user.Username,
		Email:    user.Email,
	}

	return c.Status(fiber.StatusCreated).JSON(res)
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
