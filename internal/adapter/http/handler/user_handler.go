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
	List(context.Context, int, int) ([]*user.User, int64, error)
	GetByID(context.Context, int) (*user.User, error)
	UpdateProfile(context.Context, int, string, string) error
	ChangePassword(context.Context, int, string, string) error
	ChangeRole(context.Context, int, int, string) error
	Delete(context.Context, int, int) error
	DeleteSelf(context.Context, int, string) error
}

type VerificationSender interface {
	SendVerificationForUser(context.Context, int) error
}

type UserHandler struct {
	users        UserService
	verification VerificationSender
	logger       *slog.Logger
}

func NewUserHandler(service UserService, verification VerificationSender, logger *slog.Logger) *UserHandler {
	return &UserHandler{users: service, verification: verification, logger: logger}
}

func (h *UserHandler) GetUsers(c fiber.Ctx) error {
	page, err := parsePositiveQuery(c, "page", user.DefaultPage)
	if err != nil {
		return err
	}

	limit, err := parsePositiveQuery(c, "limit", user.DefaultLimit)
	if err != nil {
		return err
	}

	if limit > user.MaxLimit {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "limit must not exceed 100"})
	}

	accounts, total, err := h.users.List(c.Context(), page, limit)
	if err != nil {
		h.logger.ErrorContext(c.Context(), "get users failed", "error", err)
		return writeDomainError(c, err)
	}

	response := make([]dto.UserSummary, 0, len(accounts))
	for _, account := range accounts {
		response = append(response, dto.UserSummary{
			ID: account.ID, Username: account.Username, Email: account.Email, Role: string(account.Role),
		})
	}

	return c.JSON(dto.UserListResponse{Data: response, Page: page, Limit: limit, Total: total})
}

func parsePositiveQuery(c fiber.Ctx, name string, fallback int) (int, error) {
	raw := c.Query(name)
	if raw == "" {
		return fallback, nil
	}

	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return 0, c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": name + " must be a positive integer"})
	}

	return value, nil
}

func (h *UserHandler) GetUserByID(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}

	account, err := h.users.GetByID(c.Context(), id)
	if err != nil {
		return writeDomainError(c, err)
	}

	return c.JSON(toUserResponse(account))
}

func (h *UserHandler) CreateUser(c fiber.Ctx) error {
	var req dto.RegisterUserRequest
	if err := bindJSON(c, &req); err != nil {
		return writeBindError(c, err)
	}

	account := user.User{Username: req.Username, Email: req.Email, Password: req.Password}
	if err := h.users.Create(c.Context(), &account); err != nil {
		return writeDomainError(c, err)
	}

	if err := h.verification.SendVerificationForUser(c.Context(), account.ID); err != nil {
		h.logger.WarnContext(c.Context(), "send verification failed", "user_id", account.ID, "error", err)
	}

	return c.Status(fiber.StatusCreated).JSON(toUserResponse(&account))
}

func (h *UserHandler) UpdateUser(c fiber.Ctx) error {
	id, err := parseID(c)
	if err != nil {
		return err
	}

	var req dto.UpdateUserRequest
	if err := bindJSON(c, &req); err != nil {
		return writeBindError(c, err)
	}

	current, err := h.users.GetByID(c.Context(), id)
	if err != nil {
		return writeDomainError(c, err)
	}

	username, email := profileValues(req, current)
	if err := h.users.UpdateProfile(c.Context(), id, username, email); err != nil {
		return writeDomainError(c, err)
	}

	if err := h.verification.SendVerificationForUser(c.Context(), id); err != nil {
		h.logger.WarnContext(c.Context(), "send verification failed", "user_id", id, "error", err)
	}

	updated, err := h.users.GetByID(c.Context(), id)
	if err != nil {
		return writeDomainError(c, err)
	}

	return c.JSON(toUserResponse(updated))
}

func (h *UserHandler) UpdateSelf(c fiber.Ctx) error {
	account, ok := currentUser(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var req dto.UpdateUserRequest
	if err := bindJSON(c, &req); err != nil {
		return writeBindError(c, err)
	}

	username, email := profileValues(req, account)
	if err := h.users.UpdateProfile(c.Context(), account.ID, username, email); err != nil {
		return writeDomainError(c, err)
	}

	if err := h.verification.SendVerificationForUser(c.Context(), account.ID); err != nil {
		h.logger.WarnContext(c.Context(), "send verification failed", "user_id", account.ID, "error", err)
	}

	updated, err := h.users.GetByID(c.Context(), account.ID)
	if err != nil {
		return writeDomainError(c, err)
	}

	return c.JSON(toUserResponse(updated))
}

func (h *UserHandler) ChangePassword(c fiber.Ctx) error {
	account, ok := currentUser(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var req dto.ChangePasswordRequest
	if err := bindJSON(c, &req); err != nil {
		return writeBindError(c, err)
	}

	if err := h.users.ChangePassword(c.Context(), account.ID, req.CurrentPassword, req.NewPassword); err != nil {
		return writeDomainError(c, err)
	}

	return c.JSON(fiber.Map{"message": "password changed; sign in again"})
}

func (h *UserHandler) ChangeRole(c fiber.Ctx) error {
	actor, ok := currentUser(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	targetID, err := parseID(c)
	if err != nil {
		return err
	}

	var req dto.ChangeRoleRequest
	if err := bindJSON(c, &req); err != nil {
		return writeBindError(c, err)
	}

	if err := h.users.ChangeRole(c.Context(), actor.ID, targetID, req.Role); err != nil {
		return writeDomainError(c, err)
	}

	return c.JSON(fiber.Map{"message": "role changed"})
}

func (h *UserHandler) DeleteUser(c fiber.Ctx) error {
	actor, ok := currentUser(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	targetID, err := parseID(c)
	if err != nil {
		return err
	}

	if err := h.users.Delete(c.Context(), actor.ID, targetID); err != nil {
		return writeDomainError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *UserHandler) DeleteSelf(c fiber.Ctx) error {
	account, ok := currentUser(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	var req dto.DeleteSelfRequest
	if err := bindJSON(c, &req); err != nil {
		return writeBindError(c, err)
	}

	if err := h.users.DeleteSelf(c.Context(), account.ID, req.CurrentPassword); err != nil {
		return writeDomainError(c, err)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func parseID(c fiber.Ctx) (int, error) {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return 0, c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid user id"})
	}

	return id, nil
}

func toUserResponse(account *user.User) dto.UserResponse {
	return dto.UserResponse{
		ID: account.ID, Username: account.Username, Email: account.Email, Role: string(account.Role),
		EmailVerified: account.EmailVerified(), PendingEmail: account.PendingEmail,
	}
}

func profileValues(req dto.UpdateUserRequest, account *user.User) (string, string) {
	username, email := account.Username, account.Email
	if req.Username != nil {
		username = *req.Username
	}

	if req.Email != nil {
		email = *req.Email
	}

	return username, email
}
