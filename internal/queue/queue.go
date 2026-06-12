package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var ErrDuplicateJob = errors.New("duplicate job")

type Job interface {
	Type() string
	Payload() any
}

type DispatchOptions struct {
	Queue     string
	ProcessAt time.Time
	MaxRetry  int
	Timeout   time.Duration
	UniqueFor time.Duration
	Retention time.Duration
	TaskID    string
}

type JobInfo struct {
	ID    string
	Queue string
}

type Dispatcher interface {
	Dispatch(ctx context.Context, job Job, options DispatchOptions) (*JobInfo, error)
}

type Worker interface {
	Run(ctx context.Context) error
}

type QueueStats struct {
	Pending   int64
	Active    int64
	Scheduled int64
	Retry     int64
	Failed    int64
	Processed int64
}

type FailedJob struct {
	ID           string
	Queue        string
	Type         string
	Retried      int
	MaxRetry     int
	LastFailedAt time.Time
	LastError    string
}

type Inspector interface {
	Queues(ctx context.Context) ([]string, error)
	Stats(ctx context.Context, queueName string) (QueueStats, error)
	Failed(ctx context.Context, queueName string, limit int) ([]FailedJob, error)
	Retry(ctx context.Context, queueName, id string) (int, error)
	Delete(ctx context.Context, queueName, id string) (int, error)
	Close() error
}

type Handler func(context.Context, json.RawMessage) error

type HandlerRegistry struct {
	handlers map[string]Handler
}

func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{handlers: make(map[string]Handler)}
}

func (r *HandlerRegistry) Register(jobType string, handler Handler) error {
	if jobType == "" {
		return errors.New("job type is required")
	}
	if handler == nil {
		return errors.New("job handler is required")
	}
	if _, exists := r.handlers[jobType]; exists {
		return fmt.Errorf("handler already registered for %q", jobType)
	}
	r.handlers[jobType] = handler
	return nil
}

func (r *HandlerRegistry) Handlers() map[string]Handler {
	handlers := make(map[string]Handler, len(r.handlers))
	for jobType, handler := range r.handlers {
		handlers[jobType] = handler
	}
	return handlers
}
