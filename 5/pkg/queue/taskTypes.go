package queue

import (
	"context"
)

// Queue интерфейс очереди
type Queue interface {
	Publish(ctx context.Context, task *Task) error
	Subscribe(ctx context.Context, handler func(*Task) error) error
	Close() error
}
