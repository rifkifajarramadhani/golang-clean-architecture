package jwt

import (
	"errors"
	"testing"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/auth"
)

func TestServiceTokenTypesVersionsAndValidation(t *testing.T) {
	service := NewService("access-secret", "refresh-secret", 15, 168, "issuer", "audience")
	access, _, err := service.GenerateAccessToken(42, 3)
	if err != nil {
		t.Fatal(err)
	}
	refresh, _, err := service.GenerateRefreshToken(42, 3)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := service.ValidateRefreshToken(refresh)
	if err != nil {
		t.Fatal(err)
	}
	if claims.UserID != 42 || claims.TokenVersion != 3 {
		t.Fatalf("claims = %+v", claims)
	}
	if _, err := service.ValidateRefreshToken(access); !errors.Is(err, auth.ErrInvalidToken) {
		t.Fatalf("access token as refresh token error = %v", err)
	}
}

func TestServiceRejectsDifferentIssuerAndAudience(t *testing.T) {
	issuerA := NewService("access-secret", "refresh-secret", 15, 168, "issuer-a", "audience")
	issuerB := NewService("access-secret", "refresh-secret", 15, 168, "issuer-b", "audience")
	token, _, err := issuerA.GenerateAccessToken(1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := issuerB.ValidateAccessToken(token); !errors.Is(err, auth.ErrInvalidToken) {
		t.Fatalf("error = %v", err)
	}
}
