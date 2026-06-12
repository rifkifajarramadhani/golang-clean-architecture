package identity

import (
	"context"
	"time"
)

const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

type Repository interface {
	CreateUser(ctx context.Context, user *User) error
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByID(ctx context.Context, id int) (*User, error)
	CreateRefreshToken(ctx context.Context, token RefreshToken) error
	RotateRefreshToken(ctx context.Context, oldHash string, replacement RefreshToken) error
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
	DeleteExpiredOrRevokedRefreshTokens(ctx context.Context, before time.Time) (int64, error)
}

type TokenService interface {
	GenerateAccessToken(userID int) (string, time.Time, error)
	GenerateRefreshToken(userID int) (string, time.Time, error)
	ValidateToken(token, expectedType string) (*TokenClaims, error)
}

type WelcomeDispatcher interface {
	DispatchWelcome(ctx context.Context, username, email string) error
}
