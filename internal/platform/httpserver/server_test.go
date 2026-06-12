package httpserver

import (
	"context"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/identity"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/platform/config"
)

type repoFake struct{}

func (repoFake) CreateUser(_ context.Context, user *identity.User) error { user.ID = 1; return nil }
func (repoFake) GetUserByEmail(context.Context, string) (*identity.User, error) {
	return nil, identity.ErrNotFound
}
func (repoFake) GetUserByID(context.Context, int) (*identity.User, error) {
	return &identity.User{ID: 1, Username: "user", Email: "user@example.com"}, nil
}
func (repoFake) CreateRefreshToken(context.Context, identity.RefreshToken) error         { return nil }
func (repoFake) RotateRefreshToken(context.Context, string, identity.RefreshToken) error { return nil }
func (repoFake) RevokeRefreshToken(context.Context, string) error                        { return nil }
func (repoFake) DeleteExpiredOrRevokedRefreshTokens(context.Context, time.Time) (int64, error) {
	return 0, nil
}

type tokensFake struct{}

func (tokensFake) GenerateAccessToken(int) (string, time.Time, error) {
	return "access", time.Now().Add(time.Minute), nil
}
func (tokensFake) GenerateRefreshToken(int) (string, time.Time, error) {
	return "refresh", time.Now().Add(time.Hour), nil
}
func (tokensFake) ValidateToken(raw, expected string) (*identity.TokenClaims, error) {
	if raw != "valid" {
		return nil, identity.ErrInvalidToken
	}
	return &identity.TokenClaims{UserID: 1, TokenType: expected}, nil
}

func TestRoutesValidationAuthenticationAndOperationalEndpoints(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := identity.NewService(repoFake{}, tokensFake{}, nil, logger)
	cfg := &config.Config{HTTP: config.HTTPConfig{BodyLimitBytes: 1024, RequestTimeoutSeconds: 1, AuthRateLimit: 10}}
	server := New(cfg, logger, nil, nil, service, tokensFake{})

	cases := []struct {
		method, path, body, auth string
		status                   int
	}{
		{"GET", "/health/live", "", "", 200},
		{"GET", "/metrics", "", "", 200},
		{"GET", "/api/v1/me", "", "", 401},
		{"GET", "/api/v1/me", "", "Bearer valid", 200},
		{"GET", "/api/v1/users", "", "Bearer valid", 404},
		{"POST", "/api/v1/auth/register", `{"username":"a","email":"bad","password":"short"}`, "", 400},
		{"POST", "/api/v1/auth/register", `{"username":"valid","email":"valid@example.com","password":"long-enough-password","admin":true}`, "", 400},
	}
	for _, test := range cases {
		req := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
		req.Header.Set("Content-Type", "application/json")
		if test.auth != "" {
			req.Header.Set("Authorization", test.auth)
		}
		res, err := server.App.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != test.status {
			t.Fatalf("%s %s status=%d want=%d", test.method, test.path, res.StatusCode, test.status)
		}
		if res.Header.Get("X-Content-Type-Options") != "nosniff" {
			t.Fatalf("%s %s missing security headers", test.method, test.path)
		}
	}
}

func TestAuthRateLimit(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := identity.NewService(repoFake{}, tokensFake{}, nil, logger)
	cfg := &config.Config{HTTP: config.HTTPConfig{BodyLimitBytes: 1024, RequestTimeoutSeconds: 1, AuthRateLimit: 1}}
	server := New(cfg, logger, nil, nil, service, tokensFake{})
	for index, want := range []int{400, 429} {
		req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(`{}`))
		req.Header.Set("Content-Type", "application/json")
		res, err := server.App.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != want {
			t.Fatalf("request %d status=%d want=%d", index, res.StatusCode, want)
		}
	}
}
