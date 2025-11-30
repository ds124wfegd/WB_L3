package queue

import (
	"fmt"
	"strings"
	"time"
)

// TaskType constants

type TaskType string

const (
	TaskTypeExpireBooking        TaskType = "expire_booking"
	TaskTypeSendNotification     TaskType = "send_notification"
	TaskTypeCleanupExpired       TaskType = "cleanup_expired"
	TaskTypeReminderNotification TaskType = "reminder_notification"
	TaskTypeEventReminder        TaskType = "event_reminder"
)

// Task represents a unit of work in the queue
type Task struct {
	ID         string                 `json:"id"`
	Type       TaskType               `json:"type"`
	Data       map[string]interface{} `json:"data"`
	ExecuteAt  time.Time              `json:"execute_at"`
	CreatedAt  time.Time              `json:"created_at"`
	Attempts   int                    `json:"attempts"`
	MaxRetries int                    `json:"max_retries"`
}

// Validate checks if the task is valid
func (t *Task) Validate() error {
	if strings.TrimSpace(t.ID) == "" {
		return fmt.Errorf("task ID is required")
	}
	if strings.TrimSpace(string(t.Type)) == "" {
		return fmt.Errorf("task type is required")
	}
	if t.Data == nil {
		t.Data = make(map[string]interface{})
	}
	return nil
}

// GetString returns a string value from task data
func (t *Task) GetString(key string) string {
	if val, ok := t.Data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

// GetInt returns an int value from task data
func (t *Task) GetInt(key string) int {
	if val, ok := t.Data[key]; ok {
		switch v := val.(type) {
		case int:
			return v
		case float64:
			return int(v)
		}
	}
	return 0
}

// GetTime returns a time value from task data
func (t *Task) GetTime(key string) time.Time {
	if val, ok := t.Data[key]; ok {
		if str, ok := val.(string); ok {
			if t, err := time.Parse(time.RFC3339, str); err == nil {
				return t
			}
		}
	}
	return time.Time{}
}
