package handler

import (
	"errors"

	"github.com/gofiber/fiber/v3"
	authdto "github.com/rifkifajarramadhani/golang-clean-architecture/internal/delivery/http/dto/auth"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/domain"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/infrastructure/logger"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/usecase"
)

type AuthHandler struct {
	authUsecase usecase.AuthUsecase
}

func NewAuthHandler(authUsecase *usecase.AuthUsecase) *AuthHandler {
	return &AuthHandler{authUsecase: *authUsecase}
}

func (h *AuthHandler) Register(c fiber.Ctx) error {
	var req authdto.RegisterRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	user := domain.User{Username: req.Username, Email: req.Email, Password: req.Password}
	if err := h.authUsecase.Register(&user); err != nil {
		switch {
		case errors.Is(err, usecase.ErrDuplicateEmail), errors.Is(err, usecase.ErrDuplicateUsername):
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		default:
			logger.Logger.Println("register failed:", err)
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to register"})
		}
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
	})
}

func (h *AuthHandler) Login(c fiber.Ctx) error {
	var req authdto.LoginRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	tokens, err := h.authUsecase.Login(req.Email, req.Password)
	if err != nil {
		if errors.Is(err, usecase.ErrInvalidCredentials) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid credentials"})
		}
		logger.Logger.Println("login failed:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to login"})
	}

	return c.JSON(authdto.AuthResponse{
		AccessToken:      tokens.AccessToken,
		AccessExpiresAt:  tokens.AccessExpiresAt.Format(timeLayout),
		RefreshToken:     tokens.RefreshToken,
		RefreshExpiresAt: tokens.RefreshExpiresAt.Format(timeLayout),
	})
}

func (h *AuthHandler) Refresh(c fiber.Ctx) error {
	var req authdto.RefreshRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	tokens, err := h.authUsecase.Refresh(req.RefreshToken)
	if err != nil {
		if errors.Is(err, usecase.ErrInvalidToken) || errors.Is(err, usecase.ErrUnauthorized) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid refresh token"})
		}
		logger.Logger.Println("refresh failed:", err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to refresh token"})
	}

	return c.JSON(authdto.AuthResponse{
		AccessToken:      tokens.AccessToken,
		AccessExpiresAt:  tokens.AccessExpiresAt.Format(timeLayout),
		RefreshToken:     tokens.RefreshToken,
		RefreshExpiresAt: tokens.RefreshExpiresAt.Format(timeLayout),
	})
}

func (h *AuthHandler) Me(c fiber.Ctx) error {
	username, ok := c.Locals("auth_username").(string)
	if !ok || username == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	user, err := h.authUsecase.Me(username)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	}

	return c.JSON(authdto.MeResponse{ID: user.ID, Username: user.Username, Email: user.Email})
}

const timeLayout = "2006-01-02T15:04:05Z07:00"
