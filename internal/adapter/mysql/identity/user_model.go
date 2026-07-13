package identity

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
