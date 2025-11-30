package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
)

// DLQHandler handles failed tasks by moving them to Dead Letter Queue
type DLQHandler interface {
	HandleFailedTask(task *Task, err error)
	GetFailedTasks(ctx context.Context, limit int) ([]*FailedTask, error)
	RequeueFailedTask(ctx context.Context, taskID string) error
	DeleteFailedTask(ctx context.Context, taskID string) error
	GetDLQStats(ctx context.Context) (*DLQStats, error)
}

// DefaultDLQHandler is the default implementation of DLQHandler
type DefaultDLQHandler struct {
	client *redis.Client
	dlq    string
}

// FailedTask represents a task that failed execution
type FailedTask struct {
	Task     *Task     `json:"task"`
	Error    string    `json:"error"`
	FailedAt time.Time `json:"failed_at"`
	Attempts int       `json:"attempts"`
}

// DLQStats contains statistics about the Dead Letter Queue
type DLQStats struct {
	TotalFailed   int64     `json:"total_failed"`
	OldestFailure time.Time `json:"oldest_failure"`
	NewestFailure time.Time `json:"newest_failure"`
	QueueSize     int64     `json:"queue_size"`
}

// NewDefaultDLQHandler creates a new DefaultDLQHandler
func NewDefaultDLQHandler(client *redis.Client, dlq string) *DefaultDLQHandler {
	return &DefaultDLQHandler{
		client: client,
		dlq:    dlq,
	}
}

// HandleFailedTask stores a failed task in the DLQ
func (d *DefaultDLQHandler) HandleFailedTask(task *Task, err error) {
	failedTask := &FailedTask{
		Task:     task,
		Error:    err.Error(),
		FailedAt: time.Now(),
		Attempts: task.Attempts,
	}

	taskData, marshalErr := json.Marshal(failedTask)
	if marshalErr != nil {
		log.Printf("Failed to marshal failed task: %v", marshalErr)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Store in DLQ with timestamp as score for sorting
	score := float64(failedTask.FailedAt.UnixNano()) / 1e9
	_, redisErr := d.client.ZAdd(ctx, d.dlq, &redis.Z{
		Score:  score,
		Member: taskData,
	}).Result()

	if redisErr != nil {
		log.Printf("Failed to send task to DLQ: %v", redisErr)
		return
	}

	log.Printf("Task %s moved to DLQ: %v", task.ID, err)
}

// GetFailedTasks retrieves failed tasks from DLQ
func (d *DefaultDLQHandler) GetFailedTasks(ctx context.Context, limit int) ([]*FailedTask, error) {
	if limit <= 0 {
		limit = 50
	}

	// Get tasks sorted by failure time (newest first)
	tasks, err := d.client.ZRevRangeByScore(ctx, d.dlq, &redis.ZRangeBy{
		Min: "-inf",
		Max: "+inf",
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get failed tasks: %v", err)
	}

	if len(tasks) > limit {
		tasks = tasks[:limit]
	}

	var failedTasks []*FailedTask
	for _, taskData := range tasks {
		var failedTask FailedTask
		if err := json.Unmarshal([]byte(taskData), &failedTask); err != nil {
			log.Printf("Failed to unmarshal failed task: %v", err)
			continue
		}
		failedTasks = append(failedTasks, &failedTask)
	}

	return failedTasks, nil
}

// RequeueFailedTask moves a failed task back to the main queue for retry
func (d *DefaultDLQHandler) RequeueFailedTask(ctx context.Context, taskID string) error {
	// Get all tasks to find the one with matching ID
	tasks, err := d.client.ZRangeByScore(ctx, d.dlq, &redis.ZRangeBy{
		Min: "-inf",
		Max: "+inf",
	}).Result()
	if err != nil {
		return fmt.Errorf("failed to get DLQ tasks: %v", err)
	}

	for _, taskData := range tasks {
		var failedTask FailedTask
		if err := json.Unmarshal([]byte(taskData), &failedTask); err != nil {
			continue
		}

		if failedTask.Task.ID == taskID {
			// Reset attempt count for retry
			failedTask.Task.Attempts = 0
			failedTask.Task.ExecuteAt = time.Now()

			// Move to main queue
			taskData, err := json.Marshal(failedTask.Task)
			if err != nil {
				return fmt.Errorf("failed to marshal task for requeue: %v", err)
			}

			pipe := d.client.Pipeline()
			pipe.LPush(ctx, "event_booking:tasks", taskData)
			pipe.ZRem(ctx, d.dlq, taskData)

			_, err = pipe.Exec(ctx)
			if err != nil {
				return fmt.Errorf("failed to requeue task: %v", err)
			}

			log.Printf("Task %s requeued from DLQ", taskID)
			return nil
		}
	}

	return fmt.Errorf("task %s not found in DLQ", taskID)
}

// DeleteFailedTask permanently removes a failed task from DLQ
func (d *DefaultDLQHandler) DeleteFailedTask(ctx context.Context, taskID string) error {
	tasks, err := d.client.ZRangeByScore(ctx, d.dlq, &redis.ZRangeBy{
		Min: "-inf",
		Max: "+inf",
	}).Result()
	if err != nil {
		return fmt.Errorf("failed to get DLQ tasks: %v", err)
	}

	for _, taskData := range tasks {
		var failedTask FailedTask
		if err := json.Unmarshal([]byte(taskData), &failedTask); err != nil {
			continue
		}

		if failedTask.Task.ID == taskID {
			if err := d.client.ZRem(ctx, d.dlq, taskData).Err(); err != nil {
				return fmt.Errorf("failed to delete task from DLQ: %v", err)
			}

			log.Printf("Task %s deleted from DLQ", taskID)
			return nil
		}
	}

	return fmt.Errorf("task %s not found in DLQ", taskID)
}

// GetDLQStats returns statistics about the DLQ
func (d *DefaultDLQHandler) GetDLQStats(ctx context.Context) (*DLQStats, error) {
	// Get total count
	count, err := d.client.ZCard(ctx, d.dlq).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get DLQ count: %v", err)
	}

	// Get oldest and newest failures
	oldestTasks, err := d.client.ZRangeByScore(ctx, d.dlq, &redis.ZRangeBy{
		Min:    "-inf",
		Max:    "+inf",
		Offset: 0,
		Count:  1,
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get oldest task: %v", err)
	}

	newestTasks, err := d.client.ZRevRangeByScore(ctx, d.dlq, &redis.ZRangeBy{
		Min:    "-inf",
		Max:    "+inf",
		Offset: 0,
		Count:  1,
	}).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get newest task: %v", err)
	}

	stats := &DLQStats{
		QueueSize: count,
	}

	// Parse oldest failure time
	if len(oldestTasks) > 0 {
		var oldestTask FailedTask
		if err := json.Unmarshal([]byte(oldestTasks[0]), &oldestTask); err == nil {
			stats.OldestFailure = oldestTask.FailedAt
		}
	}

	// Parse newest failure time
	if len(newestTasks) > 0 {
		var newestTask FailedTask
		if err := json.Unmarshal([]byte(newestTasks[0]), &newestTask); err == nil {
			stats.NewestFailure = newestTask.FailedAt
		}
	}

	return stats, nil
}

// PurgeDLQ clears all tasks from the DLQ
func (d *DefaultDLQHandler) PurgeDLQ(ctx context.Context) (int64, error) {
	count, err := d.client.ZCard(ctx, d.dlq).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get DLQ count: %v", err)
	}

	if err := d.client.Del(ctx, d.dlq).Err(); err != nil {
		return 0, fmt.Errorf("failed to purge DLQ: %v", err)
	}

	log.Printf("DLQ purged, removed %d tasks", count)
	return count, nil
}
