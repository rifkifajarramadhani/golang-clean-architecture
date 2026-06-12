package identity

import "time"

type User struct {
	ID           int
	Username     string
	Email        string
	PasswordHash string
}

type RefreshToken struct {
	UserID    int
	TokenHash string
	ExpiresAt time.Time
}

type TokenClaims struct {
	UserID    int
	TokenID   string
	TokenType string
}

type Tokens struct {
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time
}
