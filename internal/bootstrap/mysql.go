package bootstrap

import (
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/adapter/mysql/identity"
	"gorm.io/gorm"
)

func mysqlRepository(db *gorm.DB) *identity.UserRepository {
	return identity.NewUserRepository(db)
}
