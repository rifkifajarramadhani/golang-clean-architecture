package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/infrastructure/security"
)

func JWTAuth(tokenService *security.JWTService) fiber.Handler {
	return func(c fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing authorization header"})
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid authorization header"})
		}

		token := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := tokenService.ValidateToken(token, security.TokenTypeAccess)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or expired token"})
		}

		c.Locals("auth_username", claims.Subject)
		c.Locals("auth_user_id", claims.UserID)
		return c.Next()
	}
}
