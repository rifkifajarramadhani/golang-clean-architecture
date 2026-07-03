package middleware

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/auth"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/user"
)

type tokenValidatorFake struct {
	claims auth.Claims
	err    error
}

func (v tokenValidatorFake) ValidateAccessToken(string) (auth.Claims, error) {
	return v.claims, v.err
}

type userResolverFake struct {
	account *user.User
	err     error
}

func (r userResolverFake) GetByID(context.Context, int) (*user.User, error) {
	return r.account, r.err
}

func TestJWTAuthResolvesCurrentUserAndRejectsStaleOrDeletedAccounts(t *testing.T) {
	verified := time.Now()
	tests := []struct {
		name     string
		resolver userResolverFake
		claims   auth.Claims
		want     int
	}{
		{
			name:     "valid",
			resolver: userResolverFake{account: &user.User{ID: 1, TokenVersion: 2, EmailVerifiedAt: &verified}},
			claims:   auth.Claims{UserID: 1, TokenVersion: 2},
			want:     fiber.StatusOK,
		},
		{
			name:     "stale version",
			resolver: userResolverFake{account: &user.User{ID: 1, TokenVersion: 3, EmailVerifiedAt: &verified}},
			claims:   auth.Claims{UserID: 1, TokenVersion: 2},
			want:     fiber.StatusUnauthorized,
		},
		{
			name:     "deleted",
			resolver: userResolverFake{err: errors.New("not found")},
			claims:   auth.Claims{UserID: 1, TokenVersion: 2},
			want:     fiber.StatusUnauthorized,
		},
		{
			name:     "unverified",
			resolver: userResolverFake{account: &user.User{ID: 1, TokenVersion: 2}},
			claims:   auth.Claims{UserID: 1, TokenVersion: 2},
			want:     fiber.StatusUnauthorized,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			app := fiber.New()
			app.Get("/", JWTAuth(tokenValidatorFake{claims: test.claims}, test.resolver), func(c fiber.Ctx) error {
				return c.SendStatus(fiber.StatusOK)
			})
			request := httptest.NewRequest("GET", "/", nil)
			request.Header.Set("Authorization", "Bearer token")
			response, err := app.Test(request)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = response.Body.Close() }()
			if response.StatusCode != test.want {
				t.Fatalf("status = %d, want %d", response.StatusCode, test.want)
			}
		})
	}
}

func TestAdminOnly(t *testing.T) {
	tests := []struct {
		role user.Role
		want int
	}{
		{user.RoleAdmin, fiber.StatusOK},
		{user.RoleUser, fiber.StatusForbidden},
	}
	for _, test := range tests {
		app := fiber.New()
		app.Get("/", func(c fiber.Ctx) error {
			c.Locals("auth_user", &user.User{Role: test.role})
			return c.Next()
		}, AdminOnly, func(c fiber.Ctx) error {
			return c.SendStatus(fiber.StatusOK)
		})
		response, err := app.Test(httptest.NewRequest("GET", "/", nil))
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = response.Body.Close() }()
		if response.StatusCode != test.want {
			t.Fatalf("role %q status = %d, want %d", test.role, response.StatusCode, test.want)
		}
	}
}
