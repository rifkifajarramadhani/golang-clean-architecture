package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	dto "github.com/rifkifajarramadhani/golang-clean-architecture/internal/delivery/http/dto/user"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/domain"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/infrastructure/logger"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/usecase"
)

type UserHandler struct {
	userUsecase usecase.UserUsecase
}

func NewUserHandler(userUsecase *usecase.UserUsecase) *UserHandler {
	return &UserHandler{userUsecase: *userUsecase}
}

func (h *UserHandler) GetUsers(c fiber.Ctx) error {
	users, err := h.userUsecase.GetAllUsers()
	if err != nil {
		logger.Logger.Println("Error fetching users:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to get users"})
	}

	response := make([]dto.RegisterUserResponse, 0, len(users))
	for _, user := range users {
		response = append(response, dto.RegisterUserResponse{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
		})
	}

	return c.JSON(response)
}

func (h *UserHandler) GetUserByID(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user id"})
	}

	user, err := h.userUsecase.GetUserByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}

	return c.JSON(dto.RegisterUserResponse{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
	})
}

func (h *UserHandler) CreateUser(c fiber.Ctx) error {
	var req dto.RegisterUserRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	user := domain.User{Username: req.Username, Email: req.Email, Password: req.Password}
	if err := h.userUsecase.CreateUser(&user); err != nil {
		logger.Logger.Println("Error creating user:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create user"})
	}

	return c.Status(fiber.StatusCreated).JSON(dto.RegisterUserResponse{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
	})
}

func (h *UserHandler) UpdateUser(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user id"})
	}

	var req dto.UpdateUserRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	user := domain.User{ID: id, Username: req.Username, Email: req.Email, Password: req.Password}
	if err := h.userUsecase.UpdateUser(&user); err != nil {
		logger.Logger.Println("Error updating user:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update user"})
	}

	updatedUser, err := h.userUsecase.GetUserByID(id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}

	return c.JSON(dto.RegisterUserResponse{
		ID:       updatedUser.ID,
		Username: updatedUser.Username,
		Email:    updatedUser.Email,
	})
}

func (h *UserHandler) DeleteUser(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user id"})
	}

	if err := h.userUsecase.DeleteUser(id); err != nil {
		logger.Logger.Println("Error deleting user:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete user"})
	}

	return c.JSON(fiber.Map{"message": "user deleted"})
}
