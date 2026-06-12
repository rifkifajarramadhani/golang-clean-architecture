package database

import (
	"context"
	"fmt"
	"time"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/platform/config"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func NewConnection(ctx context.Context, cfg config.DatabaseConfig) (*gorm.DB, error) {
	var lastErr error
	for attempt := 1; attempt <= 10; attempt++ {
		db, err := gorm.Open(mysql.Open(cfg.DSN), &gorm.Config{})
		if err == nil {
			sqlDB, sqlErr := db.DB()
			if sqlErr == nil {
				sqlDB.SetMaxOpenConns(cfg.MaxOpenConnections)
				sqlDB.SetMaxIdleConns(cfg.MaxIdleConnections)
				sqlDB.SetConnMaxLifetime(time.Duration(cfg.ConnectionMaxMinutes) * time.Minute)
				pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
				sqlErr = sqlDB.PingContext(pingCtx)
				cancel()
			}
			if sqlErr == nil {
				return db, nil
			}
			lastErr = sqlErr
		} else {
			lastErr = err
		}
		timer := time.NewTimer(time.Duration(attempt) * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	return nil, fmt.Errorf("connect to database after retries: %w", lastErr)
}
