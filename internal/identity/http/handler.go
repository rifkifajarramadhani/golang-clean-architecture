package identityhttp

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/mail"
	"strings"
	"unicode/utf8"

	"github.com/gofiber/fiber/v3"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/identity"
)

type Handler struct{ service *identity.Service }

type authRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

func NewHandler(service *identity.Service) *Handler { return &Handler{service: service} }

func (h *Handler) Register(c fiber.Ctx) error {
	var req registerRequest
	if err := decode(c, &req); err != nil {
		return err
	}
	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if err := validateRegistration(req); err != nil {
		return err
	}
	user, err := h.service.Register(c.Context(), req.Username, req.Email, req.Password)
	if err != nil {
		return err
	}
	return c.Status(fiber.StatusCreated).JSON(userResponse(user))
}

func (h *Handler) Login(c fiber.Ctx) error {
	var req authRequest
	if err := decode(c, &req); err != nil {
		return err
	}
	if _, err := mail.ParseAddress(strings.TrimSpace(req.Email)); err != nil || req.Password == "" {
		return fmt.Errorf("%w: email and password are required", identity.ErrValidation)
	}
	if len(req.Email) > 255 || len(req.Password) > 72 {
		return fmt.Errorf("%w: credentials are too long", identity.ErrValidation)
	}
	tokens, err := h.service.Login(c.Context(), req.Email, req.Password)
	if err != nil {
		return err
	}
	return c.JSON(tokenResponse(tokens))
}

func (h *Handler) Refresh(c fiber.Ctx) error {
	var req refreshRequest
	if err := decode(c, &req); err != nil {
		return err
	}
	if strings.TrimSpace(req.RefreshToken) == "" || len(req.RefreshToken) > 4096 {
		return fmt.Errorf("%w: refresh_token is required", identity.ErrValidation)
	}
	tokens, err := h.service.Refresh(c.Context(), req.RefreshToken)
	if err != nil {
		return err
	}
	return c.JSON(tokenResponse(tokens))
}

func (h *Handler) Logout(c fiber.Ctx) error {
	var req refreshRequest
	if err := decode(c, &req); err != nil {
		return err
	}
	if err := h.service.Logout(c.Context(), req.RefreshToken); err != nil {
		return err
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) Me(c fiber.Ctx) error {
	userID, ok := c.Locals("auth_user_id").(int)
	if !ok {
		return identity.ErrUnauthorized
	}
	user, err := h.service.Me(c.Context(), userID)
	if err != nil {
		return err
	}
	return c.JSON(userResponse(user))
}

func decode(c fiber.Ctx, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(c.Body()))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("%w: invalid JSON body", identity.ErrValidation)
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fmt.Errorf("%w: request body must contain one JSON object", identity.ErrValidation)
	}
	return nil
}

func validateRegistration(req registerRequest) error {
	if utf8.RuneCountInString(req.Username) < 3 || utf8.RuneCountInString(req.Username) > 50 {
		return fmt.Errorf("%w: username must contain 3 to 50 characters", identity.ErrValidation)
	}
	if strings.ContainsAny(req.Username, " \t\r\n") {
		return fmt.Errorf("%w: username cannot contain whitespace", identity.ErrValidation)
	}
	address, err := mail.ParseAddress(req.Email)
	if err != nil || address.Address != req.Email || len(req.Email) > 255 {
		return fmt.Errorf("%w: email is invalid", identity.ErrValidation)
	}
	if len(req.Password) < 12 || len(req.Password) > 72 {
		return fmt.Errorf("%w: password must contain 12 to 72 bytes", identity.ErrValidation)
	}
	return nil
}

func userResponse(user *identity.User) fiber.Map {
	return fiber.Map{"id": user.ID, "username": user.Username, "email": user.Email}
}

func tokenResponse(tokens *identity.Tokens) fiber.Map {
	return fiber.Map{
		"access_token": tokens.AccessToken, "access_expires_at": tokens.AccessExpiresAt,
		"refresh_token": tokens.RefreshToken, "refresh_expires_at": tokens.RefreshExpiresAt,
	}
}
