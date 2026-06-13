package queueinfra

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/hibiken/asynq"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/config"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/queue"
)

type Dispatcher struct {
	client *asynq.Client
}

type RedisWorker struct {
	server *asynq.Server
	mux    *asynq.ServeMux
}

type RedisInspector struct {
	inspector *asynq.Inspector
}

func NewRedisOptions(cfg config.RedisConfig) asynq.RedisClientOpt {
	return asynq.RedisClientOpt{
		Addr:     cfg.Address,
		Password: cfg.Password,
		DB:       cfg.DB,
	}
}

func NewDispatcher(redisOptions asynq.RedisClientOpt) *Dispatcher {
	return &Dispatcher{client: asynq.NewClient(redisOptions)}
}

func (d *Dispatcher) Close() error {
	return d.client.Close()
}

func (d *Dispatcher) Dispatch(ctx context.Context, job queue.Job, options queue.DispatchOptions) (*queue.JobInfo, error) {
	payload, err := json.Marshal(job.Payload())
	if err != nil {
		return nil, fmt.Errorf("marshal job %q: %w", job.Type(), err)
	}

	asynqOptions := make([]asynq.Option, 0, 6)
	if options.Queue != "" {
		asynqOptions = append(asynqOptions, asynq.Queue(options.Queue))
	}
	if !options.ProcessAt.IsZero() {
		asynqOptions = append(asynqOptions, asynq.ProcessAt(options.ProcessAt))
	}
	if options.MaxRetry > 0 {
		asynqOptions = append(asynqOptions, asynq.MaxRetry(options.MaxRetry))
	}
	if options.Timeout > 0 {
		asynqOptions = append(asynqOptions, asynq.Timeout(options.Timeout))
	}
	if options.UniqueFor > 0 {
		asynqOptions = append(asynqOptions, asynq.Unique(options.UniqueFor))
	}
	if options.Retention > 0 {
		asynqOptions = append(asynqOptions, asynq.Retention(options.Retention))
	}
	if options.TaskID != "" {
		asynqOptions = append(asynqOptions, asynq.TaskID(options.TaskID))
	}

	info, err := d.client.EnqueueContext(ctx, asynq.NewTask(job.Type(), payload), asynqOptions...)
	if errors.Is(err, asynq.ErrDuplicateTask) || errors.Is(err, asynq.ErrTaskIDConflict) {
		return nil, queue.ErrDuplicateJob
	}
	if err != nil {
		return nil, err
	}
	return &queue.JobInfo{ID: info.ID, Queue: info.Queue}, nil
}

func NewServer(redisOptions asynq.RedisClientOpt, cfg config.QueueConfig, registry *queue.HandlerRegistry) (*asynq.Server, *asynq.ServeMux) {
	server := asynq.NewServer(redisOptions, asynq.Config{
		Concurrency:     cfg.Concurrency,
		Queues:          cfg.Queues,
		ShutdownTimeout: time.Duration(cfg.ShutdownSeconds) * time.Second,
	})
	mux := asynq.NewServeMux()
	for jobType, handler := range registry.Handlers() {
		handler := handler
		mux.HandleFunc(jobType, func(ctx context.Context, task *asynq.Task) error {
			return handler(ctx, json.RawMessage(task.Payload()))
		})
	}
	return server, mux
}

func NewRedisWorker(redisOptions asynq.RedisClientOpt, cfg config.QueueConfig, registry *queue.HandlerRegistry) *RedisWorker {
	server, mux := NewServer(redisOptions, cfg, registry)
	return &RedisWorker{server: server, mux: mux}
}

func (w *RedisWorker) Run(ctx context.Context) error {
	if err := w.server.Start(w.mux); err != nil {
		return err
	}
	<-ctx.Done()
	w.server.Shutdown()
	return nil
}

func NewRedisInspector(redisOptions asynq.RedisClientOpt) *RedisInspector {
	return &RedisInspector{inspector: asynq.NewInspector(redisOptions)}
}

func (i *RedisInspector) Close() error {
	return i.inspector.Close()
}

func (i *RedisInspector) Queues(_ context.Context) ([]string, error) {
	return i.inspector.Queues()
}

func (i *RedisInspector) Stats(_ context.Context, queueName string) (queue.QueueStats, error) {
	info, err := i.inspector.GetQueueInfo(queueName)
	if err != nil {
		return queue.QueueStats{}, err
	}
	return queue.QueueStats{
		Pending:   int64(info.Pending),
		Active:    int64(info.Active),
		Scheduled: int64(info.Scheduled),
		Retry:     int64(info.Retry),
		Failed:    int64(info.Archived),
		Processed: int64(info.ProcessedTotal),
	}, nil
}

func (i *RedisInspector) Failed(_ context.Context, queueName string, limit int) ([]queue.FailedJob, error) {
	tasks, err := i.inspector.ListArchivedTasks(queueName, asynq.PageSize(limit))
	if err != nil {
		return nil, err
	}
	failed := make([]queue.FailedJob, 0, len(tasks))
	for _, task := range tasks {
		failed = append(failed, queue.FailedJob{
			ID:           task.ID,
			Queue:        task.Queue,
			Type:         task.Type,
			Retried:      task.Retried,
			MaxRetry:     task.MaxRetry,
			LastFailedAt: task.LastFailedAt,
			LastError:    task.LastErr,
		})
	}
	return failed, nil
}

func (i *RedisInspector) Retry(_ context.Context, queueName, id string) (int, error) {
	if id == "all" {
		return i.inspector.RunAllArchivedTasks(queueName)
	}
	return 1, i.inspector.RunTask(queueName, id)
}

func (i *RedisInspector) Delete(_ context.Context, queueName, id string) (int, error) {
	if id == "all" {
		return i.inspector.DeleteAllArchivedTasks(queueName)
	}
	return 1, i.inspector.DeleteTask(queueName, id)
}
