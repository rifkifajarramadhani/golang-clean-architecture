package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/infrastructure/security"
)

func TestJWTAuthRejectsMissingToken(t *testing.T) {
	app := fiber.New()
	svc := security.NewJWTService("access-secret", "refresh-secret", 15, 24)

	app.Get("/private", JWTAuth(svc), func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/private", nil)
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.StatusCode)
	}
}

func TestJWTAuthAllowsValidAccessToken(t *testing.T) {
	app := fiber.New()
	svc := security.NewJWTService("access-secret", "refresh-secret", 15, 24)

	app.Get("/private", JWTAuth(svc), func(c fiber.Ctx) error {
		username, _ := c.Locals("auth_username").(string)
		if username != "john" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "wrong user"})
		}
		return c.JSON(fiber.Map{"ok": true})
	})

	token, _, err := svc.GenerateAccessToken("john", 1)
	if err != nil {
		t.Fatalf("failed generating token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/private", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
}
