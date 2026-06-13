package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/queue"
	"github.com/robfig/cron/v3"
)

type testJob struct{}

func (testJob) Type() string    { return "test:job" }
func (testJob) Payload() any    { return struct{}{} }
func testJobFactory() queue.Job { return testJob{} }

type parserFake struct{}

func (parserFake) Parse(expression string, location *time.Location) (Schedule, error) {
	return cron.ParseStandard("CRON_TZ=" + location.String() + " " + expression)
}

type fakeDispatcher struct {
	options []queue.DispatchOptions
	err     error
}

func (d *fakeDispatcher) Dispatch(_ context.Context, _ queue.Job, options queue.DispatchOptions) (*queue.JobInfo, error) {
	d.options = append(d.options, options)
	return &queue.JobInfo{ID: options.TaskID}, d.err
}

func TestRegistryDueUsesConfiguredTimezone(t *testing.T) {
	registry, err := NewRegistry("UTC", parserFake{})
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(Definition{
		Name:     "jakarta-midnight",
		Cron:     "0 0 * * *",
		Timezone: "Asia/Jakarta",
		Job:      testJobFactory,
	}); err != nil {
		t.Fatal(err)
	}

	due := registry.Due(time.Date(2026, 6, 10, 17, 0, 0, 0, time.UTC))
	if len(due) != 1 {
		t.Fatalf("expected one due schedule, got %d", len(due))
	}
}

func TestRunnerUsesDeterministicTaskIDAndIgnoresDuplicate(t *testing.T) {
	registry, err := NewRegistry("UTC", parserFake{})
	if err != nil {
		t.Fatal(err)
	}
	if err := registry.Register(Definition{
		Name: "cleanup",
		Cron: "* * * * *",
		Job:  testJobFactory,
	}); err != nil {
		t.Fatal(err)
	}
	dispatcher := &fakeDispatcher{err: queue.ErrDuplicateJob}
	at := time.Date(2026, 6, 11, 12, 34, 45, 0, time.UTC)

	if err := NewRunner(registry, dispatcher).Run(context.Background(), at); err != nil {
		t.Fatalf("run schedule: %v", err)
	}
	if len(dispatcher.options) != 1 {
		t.Fatalf("expected one dispatch, got %d", len(dispatcher.options))
	}
	if got, want := dispatcher.options[0].TaskID, "schedule:cleanup:20260611T1234"; got != want {
		t.Fatalf("task id = %q, want %q", got, want)
	}
	if dispatcher.options[0].Retention < 2*time.Minute {
		t.Fatalf("retention = %s, want at least 2m", dispatcher.options[0].Retention)
	}
}
