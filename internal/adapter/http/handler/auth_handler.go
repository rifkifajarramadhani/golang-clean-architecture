package handler

import (
	"context"
	"errors"
	"log/slog"

	"github.com/gofiber/fiber/v3"
	dto "github.com/rifkifajarramadhani/golang-clean-architecture/internal/adapter/http/dto/auth"
	appauth "github.com/rifkifajarramadhani/golang-clean-architecture/internal/auth"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/user"
)

type AuthService interface {
	Register(context.Context, *user.User) error
	Login(context.Context, string, string) (*appauth.Tokens, error)
	Refresh(context.Context, string) (*appauth.Tokens, error)
	Me(context.Context, string) (*user.User, error)
}

type AuthHandler struct {
	auth   AuthService
	logger *slog.Logger
}

func NewAuthHandler(service AuthService, logger *slog.Logger) *AuthHandler {
	return &AuthHandler{auth: service, logger: logger}
}

func (h *AuthHandler) Register(c fiber.Ctx) error {
	var req dto.RegisterRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	account := user.User{Username: req.Username, Email: req.Email, Password: req.Password}
	if err := h.auth.Register(c.Context(), &account); err != nil {
		if errors.Is(err, appauth.ErrDuplicateEmail) || errors.Is(err, appauth.ErrDuplicateUsername) {
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		}
		h.logger.ErrorContext(c.Context(), "register failed", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to register"})
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id": account.ID, "username": account.Username, "email": account.Email,
	})
}

func (h *AuthHandler) Login(c fiber.Ctx) error {
	var req dto.LoginRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	tokens, err := h.auth.Login(c.Context(), req.Email, req.Password)
	if err != nil {
		if errors.Is(err, appauth.ErrInvalidCredentials) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid credentials"})
		}
		h.logger.ErrorContext(c.Context(), "login failed", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to login"})
	}
	return c.JSON(toAuthResponse(tokens))
}

func (h *AuthHandler) Refresh(c fiber.Ctx) error {
	var req dto.RefreshRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	tokens, err := h.auth.Refresh(c.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, appauth.ErrInvalidToken) || errors.Is(err, appauth.ErrUnauthorized) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid refresh token"})
		}
		h.logger.ErrorContext(c.Context(), "refresh failed", "error", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to refresh token"})
	}
	return c.JSON(toAuthResponse(tokens))
}

func (h *AuthHandler) Me(c fiber.Ctx) error {
	username, ok := c.Locals("auth_username").(string)
	if !ok || username == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	account, err := h.auth.Me(c.Context(), username)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	return c.JSON(dto.MeResponse{ID: account.ID, Username: account.Username, Email: account.Email})
}

func toAuthResponse(tokens *appauth.Tokens) dto.AuthResponse {
	return dto.AuthResponse{
		AccessToken: tokens.AccessToken, AccessExpiresAt: tokens.AccessExpiresAt.Format(timeLayout),
		RefreshToken: tokens.RefreshToken, RefreshExpiresAt: tokens.RefreshExpiresAt.Format(timeLayout),
	}
}

const timeLayout = "2006-01-02T15:04:05Z07:00"
