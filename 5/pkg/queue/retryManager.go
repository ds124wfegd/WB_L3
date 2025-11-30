package queue

import (
	"math/rand"
	"strings"
	"time"
)

// RetryManager manages retry logic for failed tasks
type RetryManager struct {
	maxRetries int
	baseDelay  time.Duration
	maxDelay   time.Duration
}

// NewRetryManager creates a new RetryManager
func NewRetryManager(maxRetries int, baseDelay time.Duration) *RetryManager {
	return &RetryManager{
		maxRetries: maxRetries,
		baseDelay:  baseDelay,
		maxDelay:   baseDelay * 16, // Maximum 16x base delay
	}
}

// ShouldRetry determines if a task should be retried and returns the delay
func (r *RetryManager) ShouldRetry(task *Task, err error) (bool, time.Duration) {
	if task.Attempts >= task.MaxRetries {
		return false, 0
	}

	// Check if error is retryable
	if !r.isRetryableError(err) {
		return false, 0
	}

	// Calculate exponential backoff with jitter
	delay := r.calculateBackoff(task.Attempts)
	return true, delay
}

// isRetryableError determines if an error is retryable
func (r *RetryManager) isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Define non-retryable error patterns
	nonRetryableErrors := []string{
		"invalid",
		"not found",
		"permission denied",
		"validation failed",
	}

	errStr := err.Error()
	for _, pattern := range nonRetryableErrors {
		if strings.Contains(strings.ToLower(errStr), strings.ToLower(pattern)) {
			return false
		}
	}

	return true
}

// calculateBackoff calculates exponential backoff delay with jitter
func (r *RetryManager) calculateBackoff(attempt int) time.Duration {
	if attempt <= 0 {
		return r.baseDelay
	}

	// Exponential backoff: base * 2^(attempt-1)
	backoff := r.baseDelay * time.Duration(1<<(attempt-1))

	// Apply jitter (±25%)
	jitter := time.Duration(rand.Int63n(int64(backoff / 2)))
	if rand.Intn(2) == 0 {
		backoff += jitter
	} else {
		backoff -= jitter
	}

	// Cap at maximum delay
	if backoff > r.maxDelay {
		backoff = r.maxDelay
	}

	return backoff
}

// SetMaxRetries sets the maximum number of retries
func (r *RetryManager) SetMaxRetries(maxRetries int) {
	r.maxRetries = maxRetries
}

// SetBaseDelay sets the base delay for retries
func (r *RetryManager) SetBaseDelay(baseDelay time.Duration) {
	r.baseDelay = baseDelay
	r.maxDelay = baseDelay * 16
}
