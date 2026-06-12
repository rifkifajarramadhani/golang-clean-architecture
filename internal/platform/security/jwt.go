package security

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/identity"
)

type JWTService struct {
	accessSecret, refreshSecret []byte
	accessTTL, refreshTTL       time.Duration
	issuer, audience            string
}

type claims struct {
	TokenType string `json:"typ"`
	jwt.RegisteredClaims
}

func NewJWTService(accessSecret, refreshSecret string, accessTTL, refreshTTL time.Duration, issuer, audience string) *JWTService {
	return &JWTService{
		accessSecret: []byte(accessSecret), refreshSecret: []byte(refreshSecret),
		accessTTL: accessTTL, refreshTTL: refreshTTL, issuer: issuer, audience: audience,
	}
}

func (s *JWTService) GenerateAccessToken(userID int) (string, time.Time, error) {
	return s.generate(userID, identity.TokenTypeAccess, s.accessSecret, s.accessTTL)
}

func (s *JWTService) GenerateRefreshToken(userID int) (string, time.Time, error) {
	return s.generate(userID, identity.TokenTypeRefresh, s.refreshSecret, s.refreshTTL)
}

func (s *JWTService) ValidateToken(raw, expectedType string) (*identity.TokenClaims, error) {
	secret := s.accessSecret
	if expectedType == identity.TokenTypeRefresh {
		secret = s.refreshSecret
	}
	parsed, err := jwt.ParseWithClaims(raw, &claims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing algorithm")
		}
		return secret, nil
	}, jwt.WithIssuer(s.issuer), jwt.WithAudience(s.audience), jwt.WithExpirationRequired(), jwt.WithIssuedAt())
	if err != nil || !parsed.Valid {
		return nil, identity.ErrInvalidToken
	}
	tokenClaims, ok := parsed.Claims.(*claims)
	if !ok || tokenClaims.TokenType != expectedType {
		return nil, identity.ErrInvalidToken
	}
	userID, err := strconv.Atoi(tokenClaims.Subject)
	if err != nil || userID <= 0 {
		return nil, identity.ErrInvalidToken
	}
	return &identity.TokenClaims{UserID: userID, TokenID: tokenClaims.ID, TokenType: tokenClaims.TokenType}, nil
}

func (s *JWTService) generate(userID int, tokenType string, secret []byte, ttl time.Duration) (string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(ttl)
	tokenClaims := claims{
		TokenType: tokenType,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: s.issuer, Subject: strconv.Itoa(userID), Audience: jwt.ClaimStrings{s.audience},
			ExpiresAt: jwt.NewNumericDate(expiresAt), IssuedAt: jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now), ID: uuid.NewString(),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, tokenClaims).SignedString(secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign token: %w", err)
	}
	return token, expiresAt, nil
}
