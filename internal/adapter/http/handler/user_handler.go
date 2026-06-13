package handler

import (
	"context"
	"log/slog"
	"strconv"

	"github.com/gofiber/fiber/v3"
	dto "github.com/rifkifajarramadhani/golang-clean-architecture/internal/adapter/http/dto/user"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/user"
)

type UserService interface {
	Create(context.Context, *user.User) error
	GetAll(context.Context) ([]*user.User, error)
	GetByID(context.Context, int) (*user.User, error)
	Update(context.Context, *user.User) error
	Delete(context.Context, int) error
}

type UserHandler struct {
	users  UserService
	logger *slog.Logger
}

func NewUserHandler(service UserService, logger *slog.Logger) *UserHandler {
	return &UserHandler{users: service, logger: logger}
}

func (h *UserHandler) GetUsers(c fiber.Ctx) error {
	accounts, err := h.users.GetAll(c.Context())
	if err != nil {
		h.logger.ErrorContext(c.Context(), "get users failed", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to get users"})
	}
	response := make([]dto.RegisterUserResponse, 0, len(accounts))
	for _, account := range accounts {
		response = append(response, toUserResponse(account))
	}
	return c.JSON(response)
}

func (h *UserHandler) GetUserByID(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user id"})
	}
	account, err := h.users.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}
	return c.JSON(toUserResponse(account))
}

func (h *UserHandler) CreateUser(c fiber.Ctx) error {
	var req dto.RegisterUserRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	account := user.User{Username: req.Username, Email: req.Email, Password: req.Password}
	if err := h.users.Create(c.Context(), &account); err != nil {
		h.logger.ErrorContext(c.Context(), "create user failed", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create user"})
	}
	return c.Status(fiber.StatusCreated).JSON(toUserResponse(&account))
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
	account := user.User{ID: id, Username: req.Username, Email: req.Email, Password: req.Password}
	if err := h.users.Update(c.Context(), &account); err != nil {
		h.logger.ErrorContext(c.Context(), "update user failed", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to update user"})
	}
	updated, err := h.users.GetByID(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	}
	return c.JSON(toUserResponse(updated))
}

func (h *UserHandler) DeleteUser(c fiber.Ctx) error {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user id"})
	}
	if err := h.users.Delete(c.Context(), id); err != nil {
		h.logger.ErrorContext(c.Context(), "delete user failed", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to delete user"})
	}
	return c.JSON(fiber.Map{"message": "user deleted"})
}

func toUserResponse(account *user.User) dto.RegisterUserResponse {
	return dto.RegisterUserResponse{ID: account.ID, Username: account.Username, Email: account.Email}
}
