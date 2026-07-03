package queueinfra

import "time"

type databaseJob struct {
	ID             string
	Queue          string
	Type           string
	Payload        []byte
	Status         string
	Attempts       int
	MaxRetry       int
	TimeoutSeconds int
	RetentionSecs  int
	AvailableAt    time.Time
	LeaseToken     *string
	LeasedUntil    *time.Time
	LastError      *string
	LastFailedAt   *time.Time
	CompletedAt    *time.Time
	ExpiresAt      *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (databaseJob) TableName() string { return "queue_jobs" }

type databaseLock struct {
	LockKey   string
	JobID     string
	ExpiresAt *time.Time
	CreatedAt time.Time
}

func (databaseLock) TableName() string { return "queue_locks" }

type databaseStat struct {
	Queue          string
	ProcessedTotal int64
	UpdatedAt      time.Time
}

func (databaseStat) TableName() string { return "queue_stats" }
