package mysqladapter

import (
	"time"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/user"
)

type userModel struct {
	ID              int `gorm:"primaryKey"`
	Username        string
	Email           string `gorm:"unique"`
	Password        string
	Role            user.Role
	EmailVerifiedAt *time.Time
	PendingEmail    string
	TokenVersion    int
}

func (userModel) TableName() string { return "users" }

type refreshTokenModel struct {
	ID        int        `gorm:"primaryKey"`
	UserID    int        `gorm:"index;not null"`
	TokenHash string     `gorm:"size:64;uniqueIndex;not null"`
	ExpiresAt time.Time  `gorm:"index;not null"`
	RevokedAt *time.Time `gorm:"index"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (refreshTokenModel) TableName() string { return "refresh_tokens" }

type emailVerificationTokenModel struct {
	ID        int       `gorm:"primaryKey"`
	UserID    int       `gorm:"uniqueIndex;not null"`
	TokenHash string    `gorm:"size:64;uniqueIndex;not null"`
	ExpiresAt time.Time `gorm:"index;not null"`
	CreatedAt time.Time
}

func (emailVerificationTokenModel) TableName() string { return "email_verification_tokens" }
