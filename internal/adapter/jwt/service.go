package jwt

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/auth"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/clock"
)

const (
	tokenTypeAccess  = "access"
	tokenTypeRefresh = "refresh"
)

type tokenHeader struct {
	Algorithm string `json:"alg"`
	Type      string `json:"typ"`
}

type tokenClaims struct {
	Issuer       string `json:"iss"`
	Audience     string `json:"aud"`
	UserID       int    `json:"sub"`
	TokenVersion int    `json:"ver"`
	JTI          string `json:"jti"`
	TokenType    string `json:"typ"`
	ExpiresAt    int64  `json:"exp"`
	IssuedAt     int64  `json:"iat"`
}

type Service struct {
	accessSecret  []byte
	refreshSecret []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
	issuer        string
	audience      string
	clock         clock.Clock
}

func NewService(accessSecret, refreshSecret string, accessTTLMinutes, refreshTTLHours int, issuer, audience string) *Service {
	return &Service{
		accessSecret: []byte(accessSecret), refreshSecret: []byte(refreshSecret),
		accessTTL: time.Duration(accessTTLMinutes) * time.Minute, refreshTTL: time.Duration(refreshTTLHours) * time.Hour,
		issuer: issuer, audience: audience,
		clock: clock.Real{},
	}
}

// SetClock replaces the wall clock used to issue and validate tokens.
func (s *Service) SetClock(value clock.Clock) {
	if value != nil {
		s.clock = value
	}
}

func (s *Service) GenerateAccessToken(userID, tokenVersion int) (string, time.Time, error) {
	return s.generate(userID, tokenVersion, tokenTypeAccess, s.accessTTL, s.accessSecret)
}

func (s *Service) GenerateRefreshToken(userID, tokenVersion int) (string, time.Time, error) {
	return s.generate(userID, tokenVersion, tokenTypeRefresh, s.refreshTTL, s.refreshSecret)
}

func (s *Service) ValidateRefreshToken(token string) (auth.Claims, error) {
	return s.validate(token, tokenTypeRefresh, s.refreshSecret)
}

func (s *Service) ValidateAccessToken(token string) (auth.Claims, error) {
	return s.validate(token, tokenTypeAccess, s.accessSecret)
}

func (s *Service) generate(userID, tokenVersion int, tokenType string, ttl time.Duration, secret []byte) (string, time.Time, error) {
	now := s.clock.Now()
	jti, err := generateJTI()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("generate token identifier: %w", err)
	}
	expiresAt := now.Add(ttl)
	token, err := sign(tokenClaims{
		Issuer: s.issuer, Audience: s.audience, UserID: userID, TokenVersion: tokenVersion,
		JTI: jti, TokenType: tokenType, ExpiresAt: expiresAt.Unix(), IssuedAt: now.Unix(),
	}, secret)
	return token, expiresAt, err
}

func (s *Service) validate(token, expectedType string, secret []byte) (auth.Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return auth.Claims{}, auth.ErrInvalidToken
	}
	var header tokenHeader
	headerBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil || json.Unmarshal(headerBytes, &header) != nil || header.Algorithm != "HS256" || header.Type != "JWT" {
		return auth.Claims{}, auth.ErrInvalidToken
	}
	signingInput := parts[0] + "." + parts[1]
	providedSignature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return auth.Claims{}, auth.ErrInvalidToken
	}
	hash := hmac.New(sha256.New, secret)
	_, _ = hash.Write([]byte(signingInput))
	if !hmac.Equal(providedSignature, hash.Sum(nil)) {
		return auth.Claims{}, auth.ErrInvalidToken
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return auth.Claims{}, auth.ErrInvalidToken
	}
	var claims tokenClaims
	if json.Unmarshal(payload, &claims) != nil || claims.TokenType != expectedType ||
		claims.Issuer != s.issuer || claims.Audience != s.audience || claims.UserID <= 0 ||
		claims.TokenVersion <= 0 || claims.JTI == "" || claims.IssuedAt <= 0 ||
		claims.ExpiresAt <= claims.IssuedAt || claims.IssuedAt > s.clock.Now().Add(time.Minute).Unix() {
		return auth.Claims{}, auth.ErrInvalidToken
	}
	if s.clock.Now().Unix() >= claims.ExpiresAt {
		return auth.Claims{}, auth.ErrExpiredToken
	}
	return auth.Claims{UserID: claims.UserID, TokenVersion: claims.TokenVersion}, nil
}

func sign(claims tokenClaims, secret []byte) (string, error) {
	header, err := json.Marshal(tokenHeader{Algorithm: "HS256", Type: "JWT"})
	if err != nil {
		return "", err
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	signingInput := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
	hash := hmac.New(sha256.New, secret)
	if _, err := hash.Write([]byte(signingInput)); err != nil {
		return "", err
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(hash.Sum(nil)), nil
}

func generateJTI() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

var _ auth.TokenService = (*Service)(nil)
