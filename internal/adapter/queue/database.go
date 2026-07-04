package queueinfra

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	appclock "github.com/rifkifajarramadhani/golang-clean-architecture/internal/clock"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/config"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/queue"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	jobStatusPending   = "pending"
	jobStatusActive    = "active"
	jobStatusRetry     = "retry"
	jobStatusFailed    = "failed"
	jobStatusCompleted = "completed"
	defaultMaxRetry    = 25
	cleanupInterval    = 30 * time.Second
)

type DatabaseDispatcher struct {
	db    *gorm.DB
	clock appclock.Clock
}

type DatabaseWorker struct {
	db              *gorm.DB
	cfg             config.QueueConfig
	registry        *queue.HandlerRegistry
	weightedQueues  []string
	pollInterval    time.Duration
	reservation     time.Duration
	shutdownTimeout time.Duration
	logger          *slog.Logger
	clock           appclock.Clock

	cleanupMu   sync.Mutex
	nextCleanup time.Time
}

type DatabaseInspector struct {
	db     *gorm.DB
	queues map[string]int
	clock  appclock.Clock
}

func NewDatabaseDispatcher(db *gorm.DB) *DatabaseDispatcher {
	return &DatabaseDispatcher{db: db, clock: appclock.Real{}}
}

func (d *DatabaseDispatcher) SetClock(value appclock.Clock) {
	if value != nil {
		d.clock = value
	}
}

func (d *DatabaseDispatcher) Close() error { return nil }

func (d *DatabaseDispatcher) Dispatch(ctx context.Context, job queue.Job, options queue.DispatchOptions) (*queue.JobInfo, error) {
	payload, err := json.Marshal(job.Payload())
	if err != nil {
		return nil, fmt.Errorf("marshal job %q: %w", job.Type(), err)
	}
	queueName := options.Queue
	if queueName == "" {
		queueName = "default"
	}
	availableAt := options.ProcessAt
	if availableAt.IsZero() {
		availableAt = d.clock.Now()
	}
	maxRetry := options.MaxRetry
	if maxRetry <= 0 {
		maxRetry = defaultMaxRetry
	}
	record := databaseJob{
		ID:             chooseJobID(options.TaskID),
		Queue:          queueName,
		Type:           job.Type(),
		Payload:        payload,
		Status:         jobStatusPending,
		MaxRetry:       maxRetry,
		TimeoutSeconds: durationSeconds(options.Timeout),
		RetentionSecs:  durationSeconds(options.Retention),
		AvailableAt:    availableAt,
	}
	locks := dispatchLocks(record.ID, queueName, job.Type(), payload, options, d.clock.Now())

	err = d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, lock := range locks {
			if lock.ExpiresAt != nil {
				if err := tx.Where("lock_key = ? AND expires_at <= ?", lock.LockKey, d.clock.Now()).Delete(&databaseLock{}).Error; err != nil {
					return err
				}
			}
		}
		if err := tx.Create(&record).Error; err != nil {
			return err
		}
		for _, lock := range locks {
			if err := tx.Create(&lock).Error; err != nil {
				return err
			}
		}
		return nil
	})
	if isDuplicateKey(err) {
		return nil, queue.ErrDuplicateJob
	}
	if err != nil {
		return nil, err
	}
	return &queue.JobInfo{ID: record.ID, Queue: record.Queue}, nil
}

func NewDatabaseWorker(db *gorm.DB, cfg config.QueueConfig, registry *queue.HandlerRegistry, logger *slog.Logger) *DatabaseWorker {
	if logger == nil {
		logger = slog.Default()
	}
	weighted := weightedQueueNames(cfg.Queues)
	return &DatabaseWorker{
		db:              db,
		cfg:             cfg,
		registry:        registry,
		weightedQueues:  weighted,
		pollInterval:    time.Duration(cfg.Database.PollIntervalMilliseconds) * time.Millisecond,
		reservation:     time.Duration(cfg.Database.ReservationSeconds) * time.Second,
		shutdownTimeout: time.Duration(cfg.ShutdownSeconds) * time.Second,
		logger:          logger,
		clock:           appclock.Real{},
	}
}

func (w *DatabaseWorker) SetClock(value appclock.Clock) {
	if value != nil {
		w.clock = value
	}
}

func (w *DatabaseWorker) Run(ctx context.Context) error {
	if len(w.weightedQueues) == 0 {
		return errors.New("at least one queue must be configured")
	}
	workerCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	var workers sync.WaitGroup
	for index := 0; index < w.cfg.Concurrency; index++ {
		workers.Add(1)
		go func(offset int) {
			defer workers.Done()
			w.runLoop(workerCtx, offset)
		}(index)
	}

	<-ctx.Done()
	cancel()
	done := make(chan struct{})
	go func() {
		workers.Wait()
		close(done)
	}()
	timer := time.NewTimer(w.shutdownTimeout)
	defer timer.Stop()
	select {
	case <-done:
	case <-timer.C:
		w.logger.WarnContext(ctx, "database queue shutdown timed out")
	}
	return nil
}

func (w *DatabaseWorker) runLoop(ctx context.Context, offset int) {
	index := offset % len(w.weightedQueues)
	emptyReserves := 0
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		job, err := w.reserve(ctx, w.weightedQueues[index])
		index = (index + 1) % len(w.weightedQueues)
		if err != nil || job == nil {
			if err != nil && !errors.Is(err, context.Canceled) {
				w.logger.ErrorContext(ctx, "database queue reservation failed", "error", err)
			}
			emptyReserves++
			// Only sleep once every weighted queue slot came up empty in a
			// row; sleeping after any single empty reserve would delay ready
			// jobs on the other queues by up to a full poll cycle. Errors
			// always sleep to avoid hot-looping against a broken database.
			if err == nil && emptyReserves < len(w.weightedQueues) {
				continue
			}
			emptyReserves = 0
			timer := time.NewTimer(w.pollInterval)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
			}
			continue
		}
		emptyReserves = 0
		w.process(ctx, job)
	}
}

func (w *DatabaseWorker) reserve(ctx context.Context, queueName string) (*databaseJob, error) {
	var job databaseJob
	now := w.clock.Now()
	w.maybeCleanup(ctx, now)
	err := w.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("queue = ? AND ((status IN ? AND available_at <= ?) OR (status = ? AND leased_until <= ?))",
				queueName, []string{jobStatusPending, jobStatusRetry}, now, jobStatusActive, now).
			Order("available_at ASC, created_at ASC").
			First(&job).Error
		if err != nil {
			return err
		}
		token := uuid.NewString()
		leasedUntil := now.Add(w.reservation)
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

func (w *DatabaseWorker) process(workerCtx context.Context, job *databaseJob) {
	handler, ok := w.registry.Handlers()[job.Type]
	if !ok {
		if err := w.fail(job, fmt.Errorf("no handler registered for %q", job.Type)); err != nil {
			w.logger.ErrorContext(workerCtx, "database queue failure transition failed", "job_id", job.ID, "error", err)
		}
		return
	}

	handlerCtx := workerCtx
	cancel := func() {}
	if job.TimeoutSeconds > 0 {
		handlerCtx, cancel = context.WithTimeout(workerCtx, time.Duration(job.TimeoutSeconds)*time.Second)
	}
	defer cancel()

	heartbeatDone := make(chan struct{})
	go w.heartbeat(handlerCtx, job, heartbeatDone)
	err := runHandler(handler, handlerCtx, json.RawMessage(job.Payload))
	close(heartbeatDone)
	if err == nil {
		err = handlerCtx.Err()
	}
	if err != nil {
		if updateErr := w.fail(job, err); updateErr != nil {
			w.logger.ErrorContext(workerCtx, "database queue failure transition failed", "job_id", job.ID, "error", updateErr)
		}
		return
	}
	if err := w.complete(job); err != nil {
		w.logger.ErrorContext(workerCtx, "database queue completion failed", "job_id", job.ID, "error", err)
	}
}

func runHandler(handler queue.Handler, ctx context.Context, payload json.RawMessage) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("job handler panic: %v", recovered)
		}
	}()
	return handler(ctx, payload)
}

func (w *DatabaseWorker) heartbeat(ctx context.Context, job *databaseJob, done <-chan struct{}) {
	interval := w.reservation / 3
	if interval < time.Second {
		interval = time.Second
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-done:
			return
		case <-ticker.C:
			if err := w.db.WithContext(ctx).Model(&databaseJob{}).
				Where("id = ? AND status = ? AND lease_token = ?", job.ID, jobStatusActive, *job.LeaseToken).
				Update("leased_until", w.clock.Now().Add(w.reservation)).Error; err != nil {
				w.logger.WarnContext(ctx, "database queue heartbeat failed", "job_id", job.ID, "error", err)
			}
		}
	}
}

func (w *DatabaseWorker) complete(job *databaseJob) error {
	now := w.clock.Now()
	return w.db.Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&databaseJob{}).Where("id = ? AND status = ? AND lease_token = ?", job.ID, jobStatusActive, *job.LeaseToken)
		if job.RetentionSecs <= 0 {
			result = result.Delete(&databaseJob{})
		} else {
			expiresAt := now.Add(time.Duration(job.RetentionSecs) * time.Second)
			result = result.Updates(map[string]any{
				"status":       jobStatusCompleted,
				"completed_at": now,
				"expires_at":   expiresAt,
				"lease_token":  nil,
				"leased_until": nil,
			})
		}
		if result.Error != nil || result.RowsAffected == 0 {
			return result.Error
		}
		if job.RetentionSecs <= 0 {
			if err := tx.Where("job_id = ? AND expires_at IS NULL", job.ID).Delete(&databaseLock{}).Error; err != nil {
				return err
			}
		}
		return tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "queue"}},
			DoUpdates: clause.Assignments(map[string]any{"processed_total": gorm.Expr("processed_total + 1")}),
		}).Create(&databaseStat{Queue: job.Queue, ProcessedTotal: 1}).Error
	})
}

// maybeCleanup throttles cleanup to once per cleanupInterval across all worker
// goroutines; running it on every reserve added three queries per poll.
func (w *DatabaseWorker) maybeCleanup(ctx context.Context, now time.Time) {
	w.cleanupMu.Lock()
	if now.Before(w.nextCleanup) {
		w.cleanupMu.Unlock()
		return
	}
	w.nextCleanup = now.Add(cleanupInterval)
	w.cleanupMu.Unlock()
	w.cleanup(ctx, now)
}

func (w *DatabaseWorker) cleanup(ctx context.Context, now time.Time) {
	if err := w.db.WithContext(ctx).Where("expires_at <= ?", now).Delete(&databaseLock{}).Error; err != nil {
		w.logger.WarnContext(ctx, "database queue lock cleanup failed", "error", err)
	}
	var ids []string
	if err := w.db.WithContext(ctx).Model(&databaseJob{}).
		Where("status = ? AND expires_at <= ?", jobStatusCompleted, now).
		Limit(100).Pluck("id", &ids).Error; err != nil || len(ids) == 0 {
		if err != nil {
			w.logger.WarnContext(ctx, "database queue cleanup scan failed", "error", err)
		}
		return
	}
	if err := w.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("job_id IN ? AND expires_at IS NULL", ids).Delete(&databaseLock{}).Error; err != nil {
			return err
		}
		return tx.Where("id IN ?", ids).Delete(&databaseJob{}).Error
	}); err != nil {
		w.logger.WarnContext(ctx, "database queue completed-job cleanup failed", "error", err)
	}
}

func (w *DatabaseWorker) fail(job *databaseJob, handlerErr error) error {
	now := w.clock.Now()
	status := jobStatusRetry
	availableAt := now.Add(retryDelay(job.Attempts))
	if job.Attempts > job.MaxRetry {
		status = jobStatusFailed
		availableAt = now
	}
	message := handlerErr.Error()
	return w.db.Model(&databaseJob{}).
		Where("id = ? AND status = ? AND lease_token = ?", job.ID, jobStatusActive, *job.LeaseToken).
		Updates(map[string]any{
			"status":         status,
			"available_at":   availableAt,
			"last_error":     message,
			"last_failed_at": now,
			"lease_token":    nil,
			"leased_until":   nil,
		}).Error
}

func NewDatabaseInspector(db *gorm.DB, configuredQueues map[string]int) *DatabaseInspector {
	return &DatabaseInspector{db: db, queues: configuredQueues, clock: appclock.Real{}}
}

func (i *DatabaseInspector) SetClock(value appclock.Clock) {
	if value != nil {
		i.clock = value
	}
}

func (i *DatabaseInspector) Close() error { return nil }

func (i *DatabaseInspector) Queues(ctx context.Context) ([]string, error) {
	names := make(map[string]struct{}, len(i.queues))
	for name := range i.queues {
		names[name] = struct{}{}
	}
	var stored []string
	if err := i.db.WithContext(ctx).Model(&databaseJob{}).Distinct().Pluck("queue", &stored).Error; err != nil {
		return nil, err
	}
	for _, name := range stored {
		names[name] = struct{}{}
	}
	if err := i.db.WithContext(ctx).Model(&databaseStat{}).Distinct().Pluck("queue", &stored).Error; err != nil {
		return nil, err
	}
	for _, name := range stored {
		names[name] = struct{}{}
	}
	result := make([]string, 0, len(names))
	for name := range names {
		result = append(result, name)
	}
	sort.Strings(result)
	return result, nil
}

func (i *DatabaseInspector) Stats(ctx context.Context, queueName string) (queue.QueueStats, error) {
	now := i.clock.Now()
	var counts struct {
		Pending   int64
		Active    int64
		Scheduled int64
		Retry     int64
		Failed    int64
	}
	err := i.db.WithContext(ctx).Model(&databaseJob{}).
		Select(`COALESCE(SUM(status = ? AND available_at <= ?), 0) AS pending,
			COALESCE(SUM(status = ?), 0) AS active,
			COALESCE(SUM(status = ? AND available_at > ?), 0) AS scheduled,
			COALESCE(SUM(status = ?), 0) AS retry,
			COALESCE(SUM(status = ?), 0) AS failed`,
			jobStatusPending, now, jobStatusActive, jobStatusPending, now, jobStatusRetry, jobStatusFailed).
		Where("queue = ?", queueName).Scan(&counts).Error
	if err != nil {
		return queue.QueueStats{}, err
	}
	var stat databaseStat
	if err := i.db.WithContext(ctx).Where("queue = ?", queueName).First(&stat).Error; err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return queue.QueueStats{}, err
	}
	return queue.QueueStats{
		Pending: counts.Pending, Active: counts.Active, Scheduled: counts.Scheduled,
		Retry: counts.Retry, Failed: counts.Failed, Processed: stat.ProcessedTotal,
	}, nil
}

func (i *DatabaseInspector) Failed(ctx context.Context, queueName string, limit int) ([]queue.FailedJob, error) {
	var jobs []databaseJob
	if err := i.db.WithContext(ctx).Where("queue = ? AND status = ?", queueName, jobStatusFailed).
		Order("last_failed_at DESC").Limit(limit).Find(&jobs).Error; err != nil {
		return nil, err
	}
	result := make([]queue.FailedJob, 0, len(jobs))
	for _, job := range jobs {
		failedAt := time.Time{}
		if job.LastFailedAt != nil {
			failedAt = *job.LastFailedAt
		}
		lastError := ""
		if job.LastError != nil {
			lastError = *job.LastError
		}
		result = append(result, queue.FailedJob{
			ID: job.ID, Queue: job.Queue, Type: job.Type, Retried: max(job.Attempts-1, 0),
			MaxRetry: job.MaxRetry, LastFailedAt: failedAt, LastError: lastError,
		})
	}
	return result, nil
}

func (i *DatabaseInspector) Retry(ctx context.Context, queueName, id string) (int, error) {
	query := i.db.WithContext(ctx).Model(&databaseJob{}).Where("queue = ? AND status = ?", queueName, jobStatusFailed)
	if id != "all" {
		query = query.Where("id = ?", id)
	}
	result := query.Updates(map[string]any{
		"status": jobStatusRetry, "attempts": 0, "available_at": i.clock.Now(),
		"last_error": nil, "last_failed_at": nil,
	})
	return int(result.RowsAffected), result.Error
}

func (i *DatabaseInspector) Delete(ctx context.Context, queueName, id string) (int, error) {
	var affected int64
	err := i.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		query := tx.Model(&databaseJob{}).Where("queue = ? AND status = ?", queueName, jobStatusFailed)
		if id != "all" {
			query = query.Where("id = ?", id)
		}
		var jobs []databaseJob
		if err := query.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").Find(&jobs).Error; err != nil || len(jobs) == 0 {
			return err
		}
		ids := make([]string, 0, len(jobs))
		for _, job := range jobs {
			ids = append(ids, job.ID)
		}
		if err := tx.Where("job_id IN ?", ids).Delete(&databaseLock{}).Error; err != nil {
			return err
		}
		result := tx.Where("id IN ?", ids).Delete(&databaseJob{})
		affected = result.RowsAffected
		return result.Error
	})
	return int(affected), err
}

func chooseJobID(taskID string) string {
	if taskID != "" {
		return taskID
	}
	return uuid.NewString()
}

func dispatchLocks(jobID, queueName, jobType string, payload []byte, options queue.DispatchOptions, now time.Time) []databaseLock {
	locks := make([]databaseLock, 0, 2)
	if options.TaskID != "" {
		locks = append(locks, databaseLock{LockKey: hashLockKey("task", []byte(options.TaskID)), JobID: jobID})
	}
	if options.UniqueFor > 0 {
		expiresAt := now.Add(options.UniqueFor)
		locks = append(locks, databaseLock{
			LockKey: hashLockKey("unique", append([]byte(queueName+"\x00"+jobType+"\x00"), payload...)),
			JobID:   jobID, ExpiresAt: &expiresAt,
		})
	}
	return locks
}

func hashLockKey(prefix string, value []byte) string {
	sum := sha256.Sum256(value)
	return prefix + ":" + hex.EncodeToString(sum[:])
}

func durationSeconds(duration time.Duration) int {
	if duration <= 0 {
		return 0
	}
	seconds := int(duration / time.Second)
	if seconds == 0 {
		return 1
	}
	return seconds
}

func weightedQueueNames(queues map[string]int) []string {
	names := make([]string, 0, len(queues))
	for name := range queues {
		names = append(names, name)
	}
	sort.Strings(names)
	weighted := make([]string, 0)
	for _, name := range names {
		for range max(queues[name], 0) {
			weighted = append(weighted, name)
		}
	}
	return weighted
}

func retryDelay(attempt int) time.Duration {
	exponent := min(max(attempt-1, 0), 12)
	delay := time.Second * time.Duration(1<<exponent)
	return min(delay, time.Hour)
}

func isDuplicateKey(err error) bool {
	var mysqlErr *mysqlDriver.MySQLError
	return errors.As(err, &mysqlErr) && mysqlErr.Number == 1062
}
