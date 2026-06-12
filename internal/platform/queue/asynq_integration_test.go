//go:build integration

package queueinfra

import (
	"context"
	"os"
	"testing"

	"github.com/hibiken/asynq"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/queue"
)

type integrationJob struct{}

func (integrationJob) Type() string { return "integration:test" }
func (integrationJob) Payload() any { return integrationJob{} }

func TestRedisDispatchAndInspect(t *testing.T) {
	address := os.Getenv("REDIS_ADDRESS")
	if address == "" {
		t.Skip("REDIS_ADDRESS is not set")
	}
	options := asynq.RedisClientOpt{Addr: address}
	dispatcher := NewDispatcher(options)
	defer func() { _ = dispatcher.Close() }()
	info, err := dispatcher.Dispatch(context.Background(), integrationJob{}, queue.DispatchOptions{Queue: "integration"})
	if err != nil {
		t.Fatal(err)
	}
	inspector := NewRedisInspector(options)
	defer func() { _ = inspector.Close() }()
	t.Cleanup(func() { _ = inspector.inspector.DeleteTask("integration", info.ID) })
	stats, err := inspector.Stats(context.Background(), "integration")
	if err != nil {
		t.Fatal(err)
	}
	if stats.Pending < 1 {
		t.Fatalf("pending = %d, want at least 1 for %s", stats.Pending, info.ID)
	}
}
