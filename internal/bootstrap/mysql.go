package bootstrap

import (
	identityadapter "github.com/rifkifajarramadhani/golang-clean-architecture/internal/adapter/mysql/identity"
	"gorm.io/gorm"
)

func mysqlRepository(db *gorm.DB) *identityadapter.Repository {
	return identityadapter.NewRepository(db)
}
