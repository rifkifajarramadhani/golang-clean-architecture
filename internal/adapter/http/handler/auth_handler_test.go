package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v3"
	appauth "github.com/rifkifajarramadhani/golang-clean-architecture/internal/auth"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/user"
)

type authServiceFake struct{}

func (authServiceFake) Register(_ context.Context, account *user.User) error {
	account.ID = 42
	return nil
}

func (authServiceFake) Login(context.Context, string, string) (*appauth.Tokens, error) {
	return nil, nil
}

func (authServiceFake) Refresh(context.Context, string) (*appauth.Tokens, error) {
	return nil, nil
}

func (authServiceFake) Me(context.Context, string) (*user.User, error) {
	return nil, nil
}

func TestRegisterResponseCompatibility(t *testing.T) {
	app := fiber.New()
	app.Post("/api/auth/register", NewAuthHandler(
		authServiceFake{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	).Register)
	request := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBufferString(
		`{"username":"rifki","email":"rifki@example.com","password":"secret"}`,
	))
	request.Header.Set("Content-Type", "application/json")
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != fiber.StatusCreated {
		t.Fatalf("status = %d", response.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["id"] != float64(42) || body["username"] != "rifki" || body["email"] != "rifki@example.com" {
		t.Fatalf("response = %+v", body)
	}
}
