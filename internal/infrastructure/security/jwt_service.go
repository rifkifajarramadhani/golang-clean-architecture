package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
)

type TokenClaims struct {
	Subject   string `json:"sub"`
	UserID    int    `json:"uid"`
	JTI       string `json:"jti"`
	TokenType string `json:"typ"`
	ExpiresAt int64  `json:"exp"`
	IssuedAt  int64  `json:"iat"`
}

type JWTService struct {
	accessSecret  []byte
	refreshSecret []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
}

func NewJWTService(accessSecret, refreshSecret string, accessTTLMinutes, refreshTTLHours int) *JWTService {
	return &JWTService{
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		accessTTL:     time.Duration(accessTTLMinutes) * time.Minute,
		refreshTTL:    time.Duration(refreshTTLHours) * time.Hour,
	}
}

func (s *JWTService) GenerateAccessToken(username string, userID int) (string, time.Time, error) {
	expiresAt := time.Now().Add(s.accessTTL)
	claims := TokenClaims{
		Subject:   username,
		UserID:    userID,
		JTI:       generateJTI(),
		TokenType: TokenTypeAccess,
		ExpiresAt: expiresAt.Unix(),
		IssuedAt:  time.Now().Unix(),
	}

	token, err := s.sign(claims, s.accessSecret)
	if err != nil {
		return "", time.Time{}, err
	}

	return token, expiresAt, nil
}

func (s *JWTService) GenerateRefreshToken(username string, userID int) (string, time.Time, error) {
	expiresAt := time.Now().Add(s.refreshTTL)
	claims := TokenClaims{
		Subject:   username,
		UserID:    userID,
		JTI:       generateJTI(),
		TokenType: TokenTypeRefresh,
		ExpiresAt: expiresAt.Unix(),
		IssuedAt:  time.Now().Unix(),
	}

	token, err := s.sign(claims, s.refreshSecret)
	if err != nil {
		return "", time.Time{}, err
	}

	return token, expiresAt, nil
}

func (s *JWTService) ValidateToken(token, expectedType string) (*TokenClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}

	signingInput := parts[0] + "." + parts[1]
	providedSig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return nil, ErrInvalidToken
	}

	secret := s.accessSecret
	if expectedType == TokenTypeRefresh {
		secret = s.refreshSecret
	}

	h := hmac.New(sha256.New, secret)
	_, _ = h.Write([]byte(signingInput))
	expectedSig := h.Sum(nil)
	if !hmac.Equal(providedSig, expectedSig) {
		return nil, ErrInvalidToken
	}

	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrInvalidToken
	}

	var claims TokenClaims
	if err := json.Unmarshal(payloadBytes, &claims); err != nil {
		return nil, ErrInvalidToken
	}

	if claims.TokenType != expectedType {
		return nil, ErrInvalidToken
	}
	if time.Now().Unix() >= claims.ExpiresAt {
		return nil, ErrExpiredToken
	}

	return &claims, nil
}

func (s *JWTService) sign(claims TokenClaims, secret []byte) (string, error) {
	header := map[string]string{
		"alg": "HS256",
		"typ": "JWT",
	}

	headerBytes, err := json.Marshal(header)
	if err != nil {
		return "", err
	}

	payloadBytes, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	headerEnc := base64.RawURLEncoding.EncodeToString(headerBytes)
	payloadEnc := base64.RawURLEncoding.EncodeToString(payloadBytes)
	signingInput := fmt.Sprintf("%s.%s", headerEnc, payloadEnc)

	h := hmac.New(sha256.New, secret)
	_, _ = h.Write([]byte(signingInput))
	signature := base64.RawURLEncoding.EncodeToString(h.Sum(nil))

	return fmt.Sprintf("%s.%s", signingInput, signature), nil
}

func generateJTI() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(buf)
}
