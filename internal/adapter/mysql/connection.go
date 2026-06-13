package mysqladapter

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

const connectionAttempts = 10

func Open(ctx context.Context, dsn string, logger *slog.Logger) (*gorm.DB, error) {
	if logger == nil {
		logger = slog.Default()
	}
	var lastErr error
	for attempt := 1; attempt <= connectionAttempts; attempt++ {
		db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
		if err == nil {
			sqlDB, sqlErr := db.DB()
			if sqlErr == nil {
				err = sqlDB.PingContext(ctx)
				if err != nil {
					_ = sqlDB.Close()
				}
			} else {
				err = sqlErr
			}
		}
		if err == nil {
			return db, nil
		}
		lastErr = err
		logger.WarnContext(ctx, "database connection failed", "attempt", attempt, "error", err)
		if attempt == connectionAttempts {
			break
		}
		timer := time.NewTimer(3 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
	return nil, fmt.Errorf("connect to database after %d attempts: %w", connectionAttempts, lastErr)
}

func Close(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

func mapNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	return err
}

var ErrNotFound = errors.New("record not found")
