package router

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	appauth "github.com/rifkifajarramadhani/golang-clean-architecture/internal/auth"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/user"
)

type usersFake struct {
	verified time.Time
}

func (*usersFake) Create(context.Context, *user.User) error { return nil }
func (u *usersFake) List(context.Context, int, int) ([]*user.User, int64, error) {
	return []*user.User{{ID: 2, Role: user.RoleUser, EmailVerifiedAt: &u.verified}}, 1, nil
}
func (u *usersFake) GetByID(_ context.Context, id int) (*user.User, error) {
	role := user.Role(user.RoleUser)
	if id == 1 {
		role = user.RoleAdmin
	}
	return &user.User{
		ID: id, Username: "user_name", Email: "user@example.com", Role: role,
		TokenVersion: 1, EmailVerifiedAt: &u.verified,
	}, nil
}
func (*usersFake) UpdateProfile(context.Context, int, string, string) error  { return nil }
func (*usersFake) ChangePassword(context.Context, int, string, string) error { return nil }
func (*usersFake) ChangeRole(context.Context, int, int, string) error        { return nil }
func (*usersFake) Delete(context.Context, int, int) error                    { return nil }
func (*usersFake) DeleteSelf(context.Context, int, string) error             { return nil }

type authFake struct{}

func (authFake) Register(context.Context, *user.User) error                     { return nil }
func (authFake) Login(context.Context, string, string) (*appauth.Tokens, error) { return nil, nil }
func (authFake) Refresh(context.Context, string) (*appauth.Tokens, error)       { return nil, nil }
func (authFake) Logout(context.Context, string) error                           { return nil }
func (authFake) VerifyEmail(context.Context, string) error                      { return nil }
func (authFake) ResendVerification(context.Context, string) error               { return nil }
func (authFake) SendVerificationForUser(context.Context, int) error             { return nil }
func (authFake) Me(context.Context, int) (*user.User, error)                    { return nil, nil }

type tokensFake struct{}

func (tokensFake) ValidateAccessToken(token string) (appauth.Claims, error) {
	if token == "admin" {
		return appauth.Claims{UserID: 1, TokenVersion: 1}, nil
	}
	return appauth.Claims{UserID: 2, TokenVersion: 1}, nil
}

func TestUserRouteAuthorizationMatrix(t *testing.T) {
	app := fiber.New()
	Setup(app, &usersFake{verified: time.Now()}, authFake{}, tokensFake{}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	tests := []struct {
		name   string
		method string
		path   string
		token  string
		body   string
		want   int
	}{
		{"anonymous logout", "POST", "/api/auth/logout", "", `{"refresh_token":"token"}`, fiber.StatusOK},
		{"anonymous admin list", "GET", "/api/users", "", "", fiber.StatusUnauthorized},
		{"normal user admin list", "GET", "/api/users", "user", "", fiber.StatusForbidden},
		{"admin list", "GET", "/api/users", "admin", "", fiber.StatusOK},
		{"normal user self update", "PATCH", "/api/users/me", "user", `{"username":"new_name"}`, fiber.StatusOK},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(test.method, test.path, bytes.NewBufferString(test.body))
			if test.token != "" {
				request.Header.Set("Authorization", "Bearer "+test.token)
			}
			if test.body != "" {
				request.Header.Set("Content-Type", "application/json")
			}
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
