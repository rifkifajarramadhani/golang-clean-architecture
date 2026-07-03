package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"

	"github.com/gofiber/fiber/v3"
	appauth "github.com/rifkifajarramadhani/golang-clean-architecture/internal/auth"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/user"
)

func bindJSON(c fiber.Ctx, destination any) error {
	if !c.IsJSON() {
		return fiber.ErrUnsupportedMediaType
	}
	decoder := json.NewDecoder(bytes.NewReader(c.Body()))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fiber.ErrBadRequest
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return fiber.ErrBadRequest
	}
	return nil
}

func writeBindError(c fiber.Ctx, err error) error {
	if errors.Is(err, fiber.ErrUnsupportedMediaType) {
		return c.Status(fiber.StatusUnsupportedMediaType).JSON(fiber.Map{"error": "content type must be application/json"})
	}
	return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
}

func writeDomainError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, user.ErrInvalidInput), errors.Is(err, user.ErrInvalidRole):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, user.ErrInvalidPassword), errors.Is(err, appauth.ErrUnauthorized):
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
	case errors.Is(err, user.ErrForbidden):
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
	case errors.Is(err, appauth.ErrVerificationRate):
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "verification requested too recently"})
	case errors.Is(err, user.ErrNotFound):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "user not found"})
	case errors.Is(err, user.ErrDuplicateEmail), errors.Is(err, user.ErrDuplicateUsername), errors.Is(err, user.ErrLastAdmin):
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal server error"})
	}
}

func currentUser(c fiber.Ctx) (*user.User, bool) {
	account, ok := c.Locals("auth_user").(*user.User)
	return account, ok && account != nil
}
