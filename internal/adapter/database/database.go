// Package database exposes the selected persistence driver to process entrypoints.
package database

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	mysqladapter "github.com/rifkifajarramadhani/golang-clean-architecture/internal/adapter/mysql"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/config"
	"gorm.io/gorm"
)

func Open(ctx context.Context, cfg config.DatabaseConfig, logger *slog.Logger) (*gorm.DB, error) {
	dsn := mysqldriver.NewConfig()
	dsn.User = cfg.User
	dsn.Passwd = cfg.Password
	dsn.Net = "tcp"
	dsn.Addr = cfg.Host + ":" + strconv.Itoa(cfg.Port)
	dsn.DBName = cfg.Name
	dsn.ParseTime = true
	dsn.Collation = "utf8mb4_unicode_ci"
	dsn.Loc = time.UTC

	return mysqladapter.Open(ctx, dsn.FormatDSN(), logger)
}

func Close(db *gorm.DB) error {
	return mysqladapter.Close(db)
}
