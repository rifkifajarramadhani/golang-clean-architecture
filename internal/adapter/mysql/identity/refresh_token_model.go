package identity

import "time"

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
