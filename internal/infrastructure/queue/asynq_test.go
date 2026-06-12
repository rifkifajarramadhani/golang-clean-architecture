package queueinfra

import (
	"context"
	"testing"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/queue"
)

type invalidPayloadJob struct{}

func (invalidPayloadJob) Type() string { return "invalid" }
func (invalidPayloadJob) Payload() any { return make(chan struct{}) }

func TestDispatcherRejectsUnserializablePayloadBeforeEnqueue(t *testing.T) {
	dispatcher := &Dispatcher{}
	if _, err := dispatcher.Dispatch(context.Background(), invalidPayloadJob{}, queue.DispatchOptions{}); err == nil {
		t.Fatal("expected serialization error")
	}
}
