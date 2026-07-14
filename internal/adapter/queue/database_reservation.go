package queueinfra

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func reserveDatabaseJob(ctx context.Context, db *gorm.DB, queueName string, now time.Time, reservation time.Duration) (*databaseJob, error) {
	var job databaseJob

	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("queue = ? AND ((status IN ? AND available_at <= ?) OR (status = ? AND leased_until <= ?))",
				queueName, []string{jobStatusPending, jobStatusRetry}, now, jobStatusActive, now).
			Order("available_at ASC, created_at ASC").
			First(&job).Error; err != nil {
			return err
		}

		token := uuid.NewString()
		leasedUntil := now.Add(reservation)

		if err := tx.Model(&databaseJob{}).Where("id = ?", job.ID).Updates(map[string]any{
			"status":       jobStatusActive,
			"attempts":     gorm.Expr("attempts + 1"),
			"lease_token":  token,
			"leased_until": leasedUntil,
		}).Error; err != nil {
			return err
		}

		job.Status = jobStatusActive
		job.Attempts++
		job.LeaseToken = &token
		job.LeasedUntil = &leasedUntil

		return nil
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	return &job, err
}
