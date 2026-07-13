package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/queue"
)

type fakeInspector struct {
	retried []string
}

func (i *fakeInspector) Queues(context.Context) ([]string, error) {
	return []string{"default"}, nil
}

func (i *fakeInspector) Stats(context.Context, string) (queue.QueueStats, error) {
	return queue.QueueStats{Pending: 2, Failed: 1, Processed: 3}, nil
}

func (i *fakeInspector) Failed(context.Context, string, int) ([]queue.FailedJob, error) {
	return nil, nil
}

func (i *fakeInspector) Retry(_ context.Context, queueName, id string) (int, error) {
	i.retried = append(i.retried, queueName+":"+id)
	return 1, nil
}

func (i *fakeInspector) Delete(context.Context, string, string) (int, error) {
	return 0, nil
}

func (i *fakeInspector) Close() error { return nil }

func TestStatusUsesBackendNeutralInspector(t *testing.T) {
	inspector := &fakeInspector{}
	command := newRootCommandWithInspector(func() (queue.Inspector, error) { return inspector, nil })
	output := &bytes.Buffer{}
	command.SetOut(output)
	command.SetArgs([]string{"status"})

	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(output.String(), "default\t2\t0\t0\t0\t1\t3") {
		t.Fatalf("unexpected status output: %s", output.String())
	}
}

func TestRetryUsesBackendNeutralInspector(t *testing.T) {
	inspector := &fakeInspector{}
	command := newRootCommandWithInspector(func() (queue.Inspector, error) { return inspector, nil })
	command.SetOut(&bytes.Buffer{})
	command.SetArgs([]string{"retry", "job-1", "--queue=default"})

	if err := command.Execute(); err != nil {
		t.Fatal(err)
	}

	if len(inspector.retried) != 1 || inspector.retried[0] != "default:job-1" {
		t.Fatalf("retried = %v", inspector.retried)
	}
}
