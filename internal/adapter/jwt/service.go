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
)

const (
	tokenTypeAccess  = "access"
	tokenTypeRefresh = "refresh"
)

type tokenClaims struct {
	Subject   string `json:"sub"`
	UserID    int    `json:"uid"`
	JTI       string `json:"jti"`
	TokenType string `json:"typ"`
	ExpiresAt int64  `json:"exp"`
	IssuedAt  int64  `json:"iat"`
}

type Service struct {
	accessSecret  []byte
	refreshSecret []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
}

func NewService(accessSecret, refreshSecret string, accessTTLMinutes, refreshTTLHours int) *Service {
	return &Service{
		accessSecret: []byte(accessSecret), refreshSecret: []byte(refreshSecret),
		accessTTL:  time.Duration(accessTTLMinutes) * time.Minute,
		refreshTTL: time.Duration(refreshTTLHours) * time.Hour,
	}
}

func (s *Service) GenerateAccessToken(username string, userID int) (string, time.Time, error) {
	return s.generate(username, userID, tokenTypeAccess, s.accessTTL, s.accessSecret)
}

func (s *Service) GenerateRefreshToken(username string, userID int) (string, time.Time, error) {
	return s.generate(username, userID, tokenTypeRefresh, s.refreshTTL, s.refreshSecret)
}

func (s *Service) ValidateRefreshToken(token string) (auth.Claims, error) {
	return s.validate(token, tokenTypeRefresh, s.refreshSecret)
}

func (s *Service) ValidateAccessToken(token string) (auth.Claims, error) {
	return s.validate(token, tokenTypeAccess, s.accessSecret)
}

func (s *Service) generate(username string, userID int, tokenType string, ttl time.Duration, secret []byte) (string, time.Time, error) {
	now := time.Now()
	jti, err := generateJTI()
	if err != nil {
		return "", time.Time{}, fmt.Errorf("generate token identifier: %w", err)
	}
	expiresAt := now.Add(ttl)
	token, err := sign(tokenClaims{
		Subject: username, UserID: userID, JTI: jti, TokenType: tokenType,
		ExpiresAt: expiresAt.Unix(), IssuedAt: now.Unix(),
	}, secret)
	return token, expiresAt, err
}

func (s *Service) validate(token, expectedType string, secret []byte) (auth.Claims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
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
	if err := json.Unmarshal(payload, &claims); err != nil {
		return auth.Claims{}, auth.ErrInvalidToken
	}
	if claims.TokenType != expectedType {
		return auth.Claims{}, auth.ErrInvalidToken
	}
	if time.Now().Unix() >= claims.ExpiresAt {
		return auth.Claims{}, auth.ErrExpiredToken
	}
	return auth.Claims{Subject: claims.Subject, UserID: claims.UserID}, nil
}

func sign(claims tokenClaims, secret []byte) (string, error) {
	header, err := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
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
