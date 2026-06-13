package queueinfra

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/config"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/queue"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type integrationJob struct {
	Message string `json:"message"`
}

func (integrationJob) Type() string   { return "test:database-queue" }
func (j integrationJob) Payload() any { return j }

func TestDatabaseQueueDispatchReserveAndComplete(t *testing.T) {
	dsn := os.Getenv("QUEUE_TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("QUEUE_TEST_MYSQL_DSN is not set")
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	for _, table := range []any{&databaseJob{}, &databaseLock{}, &databaseStat{}} {
		if !db.Migrator().HasTable(table) {
			t.Fatalf("required queue table for %T is missing; apply database migrations first", table)
		}
	}

	ctx := context.Background()
	taskID := "test-database-queue-" + time.Now().Format("20060102150405.000000000")
	t.Cleanup(func() {
		db.Where("job_id = ?", taskID).Delete(&databaseLock{})
		db.Where("id = ?", taskID).Delete(&databaseJob{})
	})

	dispatcher := NewDatabaseDispatcher(db)
	options := queue.DispatchOptions{Queue: "default", TaskID: taskID, Retention: time.Minute}
	if _, err := dispatcher.Dispatch(ctx, integrationJob{Message: "hello"}, options); err != nil {
		t.Fatal(err)
	}
	if _, err := dispatcher.Dispatch(ctx, integrationJob{Message: "hello"}, options); !errors.Is(err, queue.ErrDuplicateJob) {
		t.Fatalf("duplicate dispatch error = %v, want ErrDuplicateJob", err)
	}

	registry := queue.NewHandlerRegistry()
	if err := registry.Register(integrationJob{}.Type(), func(context.Context, json.RawMessage) error { return nil }); err != nil {
		t.Fatal(err)
	}
	worker := NewDatabaseWorker(db, config.QueueConfig{
		Concurrency: 1, ShutdownSeconds: 1, Queues: map[string]int{"default": 1},
		Database: config.DatabaseQueueConfig{PollIntervalMilliseconds: 10, ReservationSeconds: 5},
	}, registry, nil)
	job, err := worker.reserve(ctx, "default")
	if err != nil {
		t.Fatal(err)
	}
	if job == nil || job.ID != taskID {
		t.Fatalf("reserved job = %#v, want %s", job, taskID)
	}
	if err := worker.complete(job); err != nil {
		t.Fatal(err)
	}

	stats, err := NewDatabaseInspector(db, map[string]int{"default": 1}).Stats(ctx, "default")
	if err != nil {
		t.Fatal(err)
	}
	if stats.Processed < 1 {
		t.Fatalf("processed = %d, want at least 1", stats.Processed)
	}

	uniqueOptions := queue.DispatchOptions{Queue: "default", UniqueFor: time.Minute}
	uniqueInfo, err := dispatcher.Dispatch(ctx, integrationJob{Message: "unique"}, uniqueOptions)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Where("job_id = ?", uniqueInfo.ID).Delete(&databaseLock{})
		db.Where("id = ?", uniqueInfo.ID).Delete(&databaseJob{})
	})
	uniqueJob, err := worker.reserve(ctx, "default")
	if err != nil {
		t.Fatal(err)
	}
	if uniqueJob == nil || uniqueJob.ID != uniqueInfo.ID {
		t.Fatalf("reserved unique job = %#v, want %s", uniqueJob, uniqueInfo.ID)
	}
	if err := worker.complete(uniqueJob); err != nil {
		t.Fatal(err)
	}
	if _, err := dispatcher.Dispatch(ctx, integrationJob{Message: "unique"}, uniqueOptions); !errors.Is(err, queue.ErrDuplicateJob) {
		t.Fatalf("post-completion unique dispatch error = %v, want ErrDuplicateJob", err)
	}
}
