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
	Logout(context.Context, string) error
	VerifyEmail(context.Context, string) error
	ResendVerification(context.Context, string) error
	SendVerificationForUser(context.Context, int) error
	Me(context.Context, int) (*user.User, error)
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
	if err := bindJSON(c, &req); err != nil {
		return writeBindError(c, err)
	}
	account := user.User{Username: req.Username, Email: req.Email, Password: req.Password}
	if err := h.auth.Register(c.Context(), &account); err != nil {
		if errors.Is(err, user.ErrInvalidInput) || errors.Is(err, user.ErrDuplicateEmail) || errors.Is(err, user.ErrDuplicateUsername) {
			return writeDomainError(c, err)
		}
		h.logger.ErrorContext(c.Context(), "register failed", "error", err)
		return writeDomainError(c, err)
	}
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id": account.ID, "username": account.Username, "email": account.Email,
		"message": "check your email to verify your account",
	})
}

func (h *AuthHandler) Login(c fiber.Ctx) error {
	var req dto.LoginRequest
	if err := bindJSON(c, &req); err != nil {
		return writeBindError(c, err)
	}
	tokens, err := h.auth.Login(c.Context(), req.Email, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, appauth.ErrInvalidCredentials):
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid credentials"})
		case errors.Is(err, appauth.ErrEmailUnverified):
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "email is not verified"})
		default:
			h.logger.ErrorContext(c.Context(), "login failed", "error", err)
			return writeDomainError(c, err)
		}
	}
	return c.JSON(toAuthResponse(tokens))
}

func (h *AuthHandler) Refresh(c fiber.Ctx) error {
	var req dto.RefreshRequest
	if err := bindJSON(c, &req); err != nil {
		return writeBindError(c, err)
	}
	tokens, err := h.auth.Refresh(c.Context(), req.RefreshToken)
	if err != nil {
		if errors.Is(err, appauth.ErrInvalidToken) || errors.Is(err, appauth.ErrUnauthorized) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid refresh token"})
		}
		h.logger.ErrorContext(c.Context(), "refresh failed", "error", err)
		return writeDomainError(c, err)
	}
	return c.JSON(toAuthResponse(tokens))
}

func (h *AuthHandler) Logout(c fiber.Ctx) error {
	var req dto.RefreshRequest
	if err := bindJSON(c, &req); err != nil {
		return writeBindError(c, err)
	}
	if err := h.auth.Logout(c.Context(), req.RefreshToken); err != nil {
		h.logger.ErrorContext(c.Context(), "logout failed", "error", err)
		return writeDomainError(c, err)
	}
	return c.JSON(fiber.Map{"message": "logged out"})
}

func (h *AuthHandler) VerifyEmail(c fiber.Ctx) error {
	var req dto.VerifyEmailRequest
	if err := bindJSON(c, &req); err != nil {
		return writeBindError(c, err)
	}
	if err := h.auth.VerifyEmail(c.Context(), req.Token); err != nil {
		if errors.Is(err, appauth.ErrInvalidToken) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid or expired verification token"})
		}
		h.logger.ErrorContext(c.Context(), "verify email failed", "error", err)
		return writeDomainError(c, err)
	}
	return c.JSON(fiber.Map{"message": "email verified"})
}

func (h *AuthHandler) ResendVerification(c fiber.Ctx) error {
	var req dto.ResendVerificationRequest
	if err := bindJSON(c, &req); err != nil {
		return writeBindError(c, err)
	}
	if err := h.auth.ResendVerification(c.Context(), req.Email); err != nil {
		h.logger.ErrorContext(c.Context(), "resend verification failed", "error", err)
	}
	return c.JSON(fiber.Map{"message": "if the account requires verification, an email has been sent"})
}

func (h *AuthHandler) Me(c fiber.Ctx) error {
	account, ok := currentUser(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}
	return c.JSON(dto.MeResponse{
		ID: account.ID, Username: account.Username, Email: account.Email, Role: string(account.Role),
		EmailVerified: account.EmailVerified(), PendingEmail: account.PendingEmail,
	})
}

func toAuthResponse(tokens *appauth.Tokens) dto.AuthResponse {
	return dto.AuthResponse{
		AccessToken: tokens.AccessToken, AccessExpiresAt: tokens.AccessExpiresAt.Format(timeLayout),
		RefreshToken: tokens.RefreshToken, RefreshExpiresAt: tokens.RefreshExpiresAt.Format(timeLayout),
	}
}

const timeLayout = "2006-01-02T15:04:05Z07:00"
