package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

const (
	defaultMaxRetries   = 3
	defaultBaseDelay    = 5 * time.Second
	defaultQueueTimeout = 5 * time.Second
	defaultBatchSize    = 10
	defaultDLQThreshold = 1000
)

// RedisQueue implements Queue interface using Redis
type RedisQueue struct {
	client          *redis.Client
	mainQueue       string
	delayedQueue    string
	processingQueue string
	dlq             string
	retryManager    *RetryManager
	dlqHandler      DLQHandler
	config          *RedisQueueConfig
	mu              sync.RWMutex
	stopChan        chan struct{}
	wg              sync.WaitGroup
	subscribers     []func(*Task) error
}

// RedisQueueConfig contains configuration for RedisQueue
type RedisQueueConfig struct {
	// Redis connection
	Addr     string
	Password string
	DB       int

	// Queue names
	MainQueue       string
	DelayedQueue    string
	ProcessingQueue string
	DLQ             string

	// Behavior
	MaxRetries    int
	BaseDelay     time.Duration
	QueueTimeout  time.Duration
	BatchSize     int
	DLQThreshold  int
	EnableDLQ     bool
	EnableMetrics bool
}

// DefaultRedisQueueConfig returns default configuration
func DefaultRedisQueueConfig() *RedisQueueConfig {
	return &RedisQueueConfig{
		Addr:            "localhost:6379",
		Password:        "",
		DB:              0,
		MainQueue:       "event_booking:tasks",
		DelayedQueue:    "event_booking:tasks:delayed",
		ProcessingQueue: "event_booking:tasks:processing",
		DLQ:             "event_booking:dlq",
		MaxRetries:      defaultMaxRetries,
		BaseDelay:       defaultBaseDelay,
		QueueTimeout:    defaultQueueTimeout,
		BatchSize:       defaultBatchSize,
		DLQThreshold:    defaultDLQThreshold,
		EnableDLQ:       true,
		EnableMetrics:   true,
	}
}

// NewRedisQueue creates a new RedisQueue instance
func NewRedisQueue(cfg *RedisQueueConfig, retryManager *RetryManager, dlqHandler DLQHandler) (*RedisQueue, error) {
	if cfg == nil {
		cfg = DefaultRedisQueueConfig()
	}

	if retryManager == nil {
		retryManager = NewRetryManager(cfg.MaxRetries, cfg.BaseDelay)
	}

	if dlqHandler == nil && cfg.EnableDLQ {
		dlqHandler = NewDefaultDLQHandler(redis.NewClient(&redis.Options{
			Addr:     cfg.Addr,
			Password: cfg.Password,
			DB:       cfg.DB,
		}), cfg.DLQ)
	}

	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 2,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %v", err)
	}

	queue := &RedisQueue{
		client:          client,
		mainQueue:       cfg.MainQueue,
		delayedQueue:    cfg.DelayedQueue,
		processingQueue: cfg.ProcessingQueue,
		dlq:             cfg.DLQ,
		retryManager:    retryManager,
		dlqHandler:      dlqHandler,
		config:          cfg,
		stopChan:        make(chan struct{}),
		subscribers:     make([]func(*Task) error, 0),
	}

	log.Printf("RedisQueue initialized: main=%s, delayed=%s, dlq=%s",
		cfg.MainQueue, cfg.DelayedQueue, cfg.DLQ)

	return queue, nil
}

// Publish sends a task to the queue
func (r *RedisQueue) Publish(ctx context.Context, task *Task) error {
	if task == nil {
		return fmt.Errorf("task cannot be nil")
	}

	// Validate and set default values
	if err := r.validateTask(task); err != nil {
		return fmt.Errorf("invalid task: %v", err)
	}

	taskData, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %v", err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Use Redis Sorted Set for delayed tasks
	if !task.ExecuteAt.IsZero() && task.ExecuteAt.After(time.Now()) {
		score := float64(task.ExecuteAt.UnixNano()) / 1e9
		_, err = r.client.ZAdd(ctx, r.delayedQueue, &redis.Z{
			Score:  score,
			Member: taskData,
		}).Result()
		if err != nil {
			return fmt.Errorf("failed to publish delayed task: %v", err)
		}

		if r.config.EnableMetrics {
			r.incrementMetric(ctx, "tasks_delayed")
		}

		log.Printf("Task %s scheduled for execution at %s", task.ID, task.ExecuteAt.Format(time.RFC3339))
	} else {
		// Use Redis List for immediate tasks
		_, err = r.client.LPush(ctx, r.mainQueue, taskData).Result()
		if err != nil {
			return fmt.Errorf("failed to publish immediate task: %v", err)
		}

		if r.config.EnableMetrics {
			r.incrementMetric(ctx, "tasks_queued")
		}

		log.Printf("Task %s published to main queue", task.ID)
	}

	return nil
}

// PublishBatch sends multiple tasks in batch
func (r *RedisQueue) PublishBatch(ctx context.Context, tasks []*Task) error {
	if len(tasks) == 0 {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	pipe := r.client.Pipeline()

	for _, task := range tasks {
		if err := r.validateTask(task); err != nil {
			log.Printf("Skipping invalid task in batch: %v", err)
			continue
		}

		taskData, err := json.Marshal(task)
		if err != nil {
			log.Printf("Failed to marshal task in batch: %v", err)
			continue
		}

		if !task.ExecuteAt.IsZero() && task.ExecuteAt.After(time.Now()) {
			score := float64(task.ExecuteAt.UnixNano()) / 1e9
			pipe.ZAdd(ctx, r.delayedQueue, &redis.Z{
				Score:  score,
				Member: taskData,
			})
		} else {
			pipe.LPush(ctx, r.mainQueue, taskData)
		}
	}

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to publish batch: %v", err)
	}

	if r.config.EnableMetrics {
		r.incrementMetricBy(ctx, "tasks_queued", int64(len(tasks)))
	}

	log.Printf("Published %d tasks in batch", len(tasks))
	return nil
}

// Subscribe starts consuming tasks from the queue
func (r *RedisQueue) Subscribe(ctx context.Context, handler func(*Task) error) error {
	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}

	r.mu.Lock()
	r.subscribers = append(r.subscribers, handler)
	r.mu.Unlock()

	// Start background processors
	r.wg.Add(3)
	go r.processDelayedTasks(ctx)
	go r.processMainQueue(ctx, handler)
	go r.monitorQueueMetrics(ctx)

	log.Println("RedisQueue subscriber started")
	return nil
}

// processMainQueue processes tasks from the main queue
func (r *RedisQueue) processMainQueue(ctx context.Context, handler func(*Task) error) {
	defer r.wg.Done()

	for {
		select {
		case <-ctx.Done():
			log.Println("Main queue processor stopped by context")
			return
		case <-r.stopChan:
			log.Println("Main queue processor stopped")
			return
		default:
			if err := r.processBatch(ctx, handler); err != nil {
				log.Printf("Error processing batch: %v", err)
				time.Sleep(time.Second) // Backoff on error
			}
		}
	}
}

// processBatch processes a batch of tasks from the main queue
func (r *RedisQueue) processBatch(ctx context.Context, handler func(*Task) error) error {
	// Move tasks from main queue to processing queue atomically
	taskData, err := r.client.BRPopLPush(ctx, r.mainQueue, r.processingQueue, r.config.QueueTimeout).Result()
	if err == redis.Nil {
		return nil // Timeout, no tasks
	}
	if err != nil {
		return fmt.Errorf("failed to move task to processing queue: %v", err)
	}

	var task Task
	if err := json.Unmarshal([]byte(taskData), &task); err != nil {
		// Move invalid task to DLQ
		log.Printf("Failed to unmarshal task: %v", err)
		r.moveToDLQ(ctx, taskData, fmt.Errorf("invalid task format: %v", err))
		return nil
	}

	// Execute task with retry logic
	if err := r.executeTaskWithRetry(ctx, &task, handler); err != nil {
		log.Printf("Task %s failed after %d attempts: %v", task.ID, task.Attempts, err)
		if r.dlqHandler != nil {
			r.dlqHandler.HandleFailedTask(&task, err)
		}
	} else {
		log.Printf("Task %s completed successfully", task.ID)
	}

	// Remove from processing queue regardless of outcome
	if err := r.client.LRem(ctx, r.processingQueue, 1, taskData).Err(); err != nil {
		log.Printf("Failed to remove task from processing queue: %v", err)
	}

	return nil
}

// processDelayedTasks moves ready delayed tasks to main queue
func (r *RedisQueue) processDelayedTasks(ctx context.Context) {
	defer r.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Delayed tasks processor stopped by context")
			return
		case <-r.stopChan:
			log.Println("Delayed tasks processor stopped")
			return
		case <-ticker.C:
			if err := r.moveReadyDelayedTasks(ctx); err != nil {
				log.Printf("Failed to process delayed tasks: %v", err)
			}
		}
	}
}

// moveReadyDelayedTasks moves ready delayed tasks to main queue
func (r *RedisQueue) moveReadyDelayedTasks(ctx context.Context) error {
	now := float64(time.Now().UnixNano()) / 1e9

	// Get tasks that are ready to execute
	tasks, err := r.client.ZRangeByScore(ctx, r.delayedQueue, &redis.ZRangeBy{
		Min: "0",
		Max: fmt.Sprintf("%f", now),
	}).Result()
	if err != nil {
		return fmt.Errorf("failed to get delayed tasks: %v", err)
	}

	if len(tasks) == 0 {
		return nil
	}

	// Move to main queue in batch
	pipe := r.client.Pipeline()
	for _, taskData := range tasks {
		pipe.LPush(ctx, r.mainQueue, taskData)
	}
	pipe.ZRemRangeByScore(ctx, r.delayedQueue, "0", fmt.Sprintf("%f", now))

	_, err = pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to move delayed tasks: %v", err)
	}

	if r.config.EnableMetrics {
		r.incrementMetricBy(ctx, "tasks_delayed_processed", int64(len(tasks)))
	}

	log.Printf("Moved %d delayed tasks to main queue", len(tasks))
	return nil
}

// executeTaskWithRetry executes a task with retry logic
func (r *RedisQueue) executeTaskWithRetry(ctx context.Context, task *Task, handler func(*Task) error) error {
	for {
		task.Attempts++
		startTime := time.Now()

		err := handler(task)
		if err == nil {
			if r.config.EnableMetrics {
				r.recordTaskSuccess(ctx, task, time.Since(startTime))
			}
			return nil // Success
		}

		if r.config.EnableMetrics {
			r.recordTaskFailure(ctx, task, err, time.Since(startTime))
		}

		// Check if we should retry
		shouldRetry, delay := r.retryManager.ShouldRetry(task, err)
		if !shouldRetry {
			return err // Final failure
		}

		log.Printf("Task %s failed (attempt %d/%d), retrying in %v: %v",
			task.ID, task.Attempts, task.MaxRetries, delay, err)

		// Wait before retry with jitter
		jitteredDelay := delay + time.Duration(rand.Int63n(int64(delay/time.Millisecond)))*time.Millisecond
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(jitteredDelay):
			// Continue to next attempt
		}
	}
}

// moveToDLQ moves a failed task to Dead Letter Queue
func (r *RedisQueue) moveToDLQ(ctx context.Context, taskData string, err error) {
	if !r.config.EnableDLQ || r.dlqHandler == nil {
		return
	}

	var task Task
	if jsonErr := json.Unmarshal([]byte(taskData), &task); jsonErr != nil {
		// If we can't unmarshal, store the raw data
		failedTask := &Task{
			ID:        fmt.Sprintf("corrupted_%d", time.Now().UnixNano()),
			Type:      "corrupted",
			Data:      map[string]interface{}{"raw_data": taskData},
			CreatedAt: time.Now(),
		}
		r.dlqHandler.HandleFailedTask(failedTask, fmt.Errorf("corrupted task: %v", jsonErr))
	} else {
		r.dlqHandler.HandleFailedTask(&task, err)
	}

	if r.config.EnableMetrics {
		r.incrementMetric(ctx, "tasks_dlq")
	}
}

// validateTask validates task structure and sets defaults
func (r *RedisQueue) validateTask(task *Task) error {
	if task.ID == "" {
		task.ID = generateTaskID()
	}
	if task.Type == "" {
		return fmt.Errorf("task type is required")
	}
	if task.Data == nil {
		task.Data = make(map[string]interface{})
	}
	if task.MaxRetries == 0 {
		task.MaxRetries = r.config.MaxRetries
	}
	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now()
	}
	if task.ExecuteAt.IsZero() {
		task.ExecuteAt = time.Now()
	}

	return nil
}

// monitorQueueMetrics monitors queue metrics and health
func (r *RedisQueue) monitorQueueMetrics(ctx context.Context) {
	defer r.wg.Done()

	if !r.config.EnableMetrics {
		return
	}

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-r.stopChan:
			return
		case <-ticker.C:
			r.collectQueueMetrics(ctx)
		}
	}
}

// collectQueueMetrics collects various queue metrics
func (r *RedisQueue) collectQueueMetrics(ctx context.Context) {
	pipe := r.client.Pipeline()

	mainLen := pipe.LLen(ctx, r.mainQueue)
	delayedLen := pipe.ZCard(ctx, r.delayedQueue)
	processingLen := pipe.LLen(ctx, r.processingQueue)
	dlqLen := pipe.LLen(ctx, r.dlq)

	_, err := pipe.Exec(ctx)
	if err != nil {
		log.Printf("Failed to collect queue metrics: %v", err)
		return
	}

	metrics := map[string]interface{}{
		"queue_main_len":       mainLen.Val(),
		"queue_delayed_len":    delayedLen.Val(),
		"queue_processing_len": processingLen.Val(),
		"queue_dlq_len":        dlqLen.Val(),
		"timestamp":            time.Now().Unix(),
	}

	// Store metrics in Redis
	metricsData, err := json.Marshal(metrics)
	if err == nil {
		r.client.Set(ctx, "event_booking:queue:metrics", metricsData, 2*time.Minute)
	}

	// Log if queues are getting too large
	if mainLen.Val() > int64(r.config.DLQThreshold) {
		log.Printf("WARNING: Main queue size (%d) exceeds threshold (%d)",
			mainLen.Val(), r.config.DLQThreshold)
	}
}

// incrementMetric increments a counter metric
func (r *RedisQueue) incrementMetric(ctx context.Context, metric string) {
	if !r.config.EnableMetrics {
		return
	}

	key := fmt.Sprintf("event_booking:metrics:%s", metric)
	r.client.Incr(ctx, key)
	r.client.Expire(ctx, key, 24*time.Hour)
}

// incrementMetricBy increments a counter metric by specific value
func (r *RedisQueue) incrementMetricBy(ctx context.Context, metric string, value int64) {
	if !r.config.EnableMetrics {
		return
	}

	key := fmt.Sprintf("event_booking:metrics:%s", metric)
	r.client.IncrBy(ctx, key, value)
	r.client.Expire(ctx, key, 24*time.Hour)
}

// recordTaskSuccess records successful task execution metrics
func (r *RedisQueue) recordTaskSuccess(ctx context.Context, task *Task, duration time.Duration) {
	r.incrementMetric(ctx, "tasks_success")
	r.incrementMetric(ctx, fmt.Sprintf("tasks_success_%s", task.Type))

	// Record execution time
	durationKey := fmt.Sprintf("event_booking:metrics:task_duration_%s", task.Type)
	r.client.HIncrBy(ctx, "event_booking:metrics:task_timing", durationKey, int64(duration.Milliseconds()))
}

// recordTaskFailure records failed task execution metrics
func (r *RedisQueue) recordTaskFailure(ctx context.Context, task *Task, err error, duration time.Duration) {
	r.incrementMetric(ctx, "tasks_failure")
	r.incrementMetric(ctx, fmt.Sprintf("tasks_failure_%s", task.Type))

	// Record error type
	errorType := "unknown"
	if err != nil {
		errorType = "generic"
	}
	r.incrementMetric(ctx, fmt.Sprintf("errors_%s", errorType))
}

// GetQueueStats returns current queue statistics
func (r *RedisQueue) GetQueueStats(ctx context.Context) (*QueueStats, error) {
	pipe := r.client.Pipeline()

	mainLen := pipe.LLen(ctx, r.mainQueue)
	delayedLen := pipe.ZCard(ctx, r.delayedQueue)
	processingLen := pipe.LLen(ctx, r.processingQueue)
	dlqLen := pipe.LLen(ctx, r.dlq)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get queue stats: %v", err)
	}

	return &QueueStats{
		MainQueue:       mainLen.Val(),
		DelayedQueue:    delayedLen.Val(),
		ProcessingQueue: processingLen.Val(),
		DLQ:             dlqLen.Val(),
		Timestamp:       time.Now(),
	}, nil
}

// Purge clears all queues (use with caution!)
func (r *RedisQueue) Purge(ctx context.Context) error {
	pipe := r.client.Pipeline()

	pipe.Del(ctx, r.mainQueue)
	pipe.Del(ctx, r.delayedQueue)
	pipe.Del(ctx, r.processingQueue)
	pipe.Del(ctx, r.dlq)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to purge queues: %v", err)
	}

	log.Println("All queues purged")
	return nil
}

// Close gracefully shuts down the queue
func (r *RedisQueue) Close() error {
	close(r.stopChan)
	r.wg.Wait()

	if err := r.client.Close(); err != nil {
		return fmt.Errorf("failed to close Redis client: %v", err)
	}

	log.Println("RedisQueue closed successfully")
	return nil
}

// HealthCheck performs a health check on the queue
func (r *RedisQueue) HealthCheck(ctx context.Context) error {
	if err := r.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis connection failed: %v", err)
	}

	// Check if we can perform basic operations
	if err := r.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis operation failed: %v", err)
	}

	return nil
}

// QueueStats contains statistics about queue state
type QueueStats struct {
	MainQueue       int64     `json:"main_queue"`
	DelayedQueue    int64     `json:"delayed_queue"`
	ProcessingQueue int64     `json:"processing_queue"`
	DLQ             int64     `json:"dlq"`
	Timestamp       time.Time `json:"timestamp"`
}

// generateTaskID generates a unique task ID
func generateTaskID() string {
	return fmt.Sprintf("task_%d_%d", time.Now().UnixNano(), rand.Int63())
}
