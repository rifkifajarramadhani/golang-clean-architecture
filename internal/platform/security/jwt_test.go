package security

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/identity"
)

func TestJWTValidatesTypeAudienceAndAlgorithm(t *testing.T) {
	service := NewJWTService("access-secret-at-least-thirty-two-characters", "refresh-secret-at-least-thirty-two-characters", time.Minute, time.Hour, "issuer", "audience")
	raw, _, err := service.GenerateAccessToken(42)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := service.ValidateToken(raw, identity.TokenTypeAccess)
	if err != nil || claims.UserID != 42 {
		t.Fatalf("claims=%+v err=%v", claims, err)
	}
	if _, err := service.ValidateToken(raw, identity.TokenTypeRefresh); !errors.Is(err, identity.ErrInvalidToken) {
		t.Fatalf("type mismatch error = %v", err)
	}
	none := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{"sub": "42", "typ": "access"})
	noneRaw, _ := none.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if _, err := service.ValidateToken(noneRaw, identity.TokenTypeAccess); !errors.Is(err, identity.ErrInvalidToken) {
		t.Fatalf("algorithm error = %v", err)
	}
}

func TestJWTRejectsExpiredToken(t *testing.T) {
	service := NewJWTService("access-secret-at-least-thirty-two-characters", "refresh-secret-at-least-thirty-two-characters", -time.Second, time.Hour, "issuer", "audience")
	raw, _, err := service.GenerateAccessToken(42)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ValidateToken(raw, identity.TokenTypeAccess); !errors.Is(err, identity.ErrInvalidToken) {
		t.Fatalf("expired token error = %v", err)
	}
}
