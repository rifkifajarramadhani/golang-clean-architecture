package middleware

import (
	"context"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/auth"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/user"
)

type AccessTokenValidator interface {
	ValidateAccessToken(string) (auth.Claims, error)
}

type UserResolver interface {
	GetByID(context.Context, int) (*user.User, error)
}

func JWTAuth(tokens AccessTokenValidator, users UserResolver) fiber.Handler {
	return func(c fiber.Ctx) error {
		parts := strings.Fields(c.Get("Authorization"))
		if len(parts) != 2 || parts[0] != "Bearer" || parts[1] == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid authorization header"})
		}
		claims, err := tokens.ValidateAccessToken(parts[1])
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or expired token"})
		}
		account, err := users.GetByID(c.Context(), claims.UserID)
		if err != nil || account == nil || account.TokenVersion != claims.TokenVersion || !account.EmailVerified() {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "unauthorized"})
		}
		c.Locals("auth_user", account)
		return c.Next()
	}
}

func AdminOnly(c fiber.Ctx) error {
	account, ok := c.Locals("auth_user").(*user.User)
	if !ok || account == nil || !account.IsAdmin() {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "forbidden"})
	}
	return c.Next()
}
