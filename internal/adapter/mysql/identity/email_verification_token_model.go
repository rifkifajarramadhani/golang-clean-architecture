package identity

import "time"

type emailVerificationTokenModel struct {
	ID        int       `gorm:"primaryKey"`
	UserID    int       `gorm:"uniqueIndex;not null"`
	TokenHash string    `gorm:"size:64;uniqueIndex;not null"`
	ExpiresAt time.Time `gorm:"index;not null"`
	CreatedAt time.Time
}

func (emailVerificationTokenModel) TableName() string { return "email_verification_tokens" }
