package jwt

import (
	"errors"
	"testing"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/auth"
)

func TestServiceTokenTypesAndValidation(t *testing.T) {
	service := NewService("access-secret", "refresh-secret", 15, 168)
	access, _, err := service.GenerateAccessToken("rifki", 42)
	if err != nil {
		t.Fatal(err)
	}
	refresh, _, err := service.GenerateRefreshToken("rifki", 42)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := service.ValidateRefreshToken(refresh)
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != "rifki" || claims.UserID != 42 {
		t.Fatalf("claims = %+v", claims)
	}
	if _, err := service.ValidateRefreshToken(access); !errors.Is(err, auth.ErrInvalidToken) {
		t.Fatalf("access token as refresh token error = %v", err)
	}
}
