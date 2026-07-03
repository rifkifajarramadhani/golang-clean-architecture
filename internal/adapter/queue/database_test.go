package queueinfra

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/queue"
)

func TestWeightedQueueNamesAreStable(t *testing.T) {
	got := weightedQueueNames(map[string]int{"maintenance": 1, "default": 3, "ignored": 0})
	want := []string{"default", "default", "default", "maintenance"}
	if len(got) != len(want) {
		t.Fatalf("weighted queues = %v, want %v", got, want)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("weighted queues = %v, want %v", got, want)
		}
	}
}

func TestRunHandlerConvertsPanicToError(t *testing.T) {
	err := runHandler(func(context.Context, json.RawMessage) error {
		panic("boom")
	}, context.Background(), nil)
	if err == nil {
		t.Fatal("expected panic to become an error")
	}
}

func TestRetryDelayUsesCappedExponentialBackoff(t *testing.T) {
	if got := retryDelay(1); got != time.Second {
		t.Fatalf("first retry delay = %s, want 1s", got)
	}
	if got := retryDelay(100); got != time.Hour {
		t.Fatalf("capped retry delay = %s, want 1h", got)
	}
}

func TestDispatchLocksPreserveTaskAndUniqueSemantics(t *testing.T) {
	options := queue.DispatchOptions{TaskID: "scheduled-task", UniqueFor: time.Minute}
	now := time.Now()
	first := dispatchLocks("job-one", "default", "demo", []byte(`{"message":"hello"}`), options, now)
	second := dispatchLocks("job-two", "default", "demo", []byte(`{"message":"hello"}`), options, now)

	if len(first) != 2 || len(second) != 2 {
		t.Fatalf("locks = %d and %d, want 2 each", len(first), len(second))
	}
	if first[0].LockKey != second[0].LockKey || first[1].LockKey != second[1].LockKey {
		t.Fatal("equivalent jobs produced different lock keys")
	}
	if first[0].ExpiresAt != nil {
		t.Fatal("task ID lock must not expire while the job exists")
	}
	if first[1].ExpiresAt == nil {
		t.Fatal("unique lock must expire")
	}
}
