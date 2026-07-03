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

// VerificationRolePolicy decides the verified user's role while the
// verification transaction still holds its consistency locks.
type VerificationRolePolicy func(user.User, int64) (user.Role, error)
