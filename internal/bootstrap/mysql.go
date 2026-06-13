package bootstrap

import (
	mysqladapter "github.com/rifkifajarramadhani/golang-clean-architecture/internal/adapter/mysql"
	"gorm.io/gorm"
)

func mysqlRepository(db *gorm.DB) *mysqladapter.UserRepository {
	return mysqladapter.NewUserRepository(db)
}
