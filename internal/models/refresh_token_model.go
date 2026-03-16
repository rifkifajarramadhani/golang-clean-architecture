package models

import "time"

type RefreshToken struct {
	ID        int        `gorm:"primaryKey"`
	UserID    int        `gorm:"index;not null"`
	TokenHash string     `gorm:"size:64;uniqueIndex;not null"`
	ExpiresAt time.Time  `gorm:"index;not null"`
	RevokedAt *time.Time `gorm:"index"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
