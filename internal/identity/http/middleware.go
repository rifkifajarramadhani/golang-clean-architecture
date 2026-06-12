package identityhttp

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/identity"
)

func Authenticate(tokens identity.TokenService) fiber.Handler {
	return func(c fiber.Ctx) error {
		header := c.Get(fiber.HeaderAuthorization)
		if !strings.HasPrefix(header, "Bearer ") {
			return identity.ErrUnauthorized
		}
		claims, err := tokens.ValidateToken(strings.TrimPrefix(header, "Bearer "), identity.TokenTypeAccess)
		if err != nil {
			return identity.ErrUnauthorized
		}
		c.Locals("auth_user_id", claims.UserID)
		return c.Next()
	}
}
