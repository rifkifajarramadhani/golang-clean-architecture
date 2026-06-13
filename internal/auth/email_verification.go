package auth

import (
	"time"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/user"
)

type EmailVerificationToken struct {
	ID        int
	UserID    int
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
}

type EmailVerificationResult struct {
	User              *user.User
	FirstVerification bool
}
