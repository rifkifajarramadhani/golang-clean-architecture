// Package identity is the MySQL adapter for the user and auth domains. A single
// Repository backs both because registration and email verification span the
// users, refresh-token, and email-verification tables in one transaction.
package identity

import (
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/auth"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/user"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

var (
	_ auth.UserStore              = (*Repository)(nil)
	_ auth.RefreshTokenRepository = (*Repository)(nil)
	_ auth.VerificationRepository = (*Repository)(nil)
	_ auth.MaintenanceRepository  = (*Repository)(nil)
	_ user.Repository             = (*Repository)(nil)
)
