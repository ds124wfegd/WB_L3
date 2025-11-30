package service

import (
	"context"

	"github.com/ds124wfegd/WB_L3/5/pkg/queue"
)

// QueueAdapter адаптирует queue.Queue к TaskPublisher интерфейсу
type QueueAdapter struct {
	queue queue.Queue
}

// NewQueueAdapter создает новый адаптер для очереди
func NewQueueAdapter(q queue.Queue) *QueueAdapter {
	return &QueueAdapter{queue: q}
}

// Publish публикует задачу, преобразуя service.Task в queue.Task
func (a *QueueAdapter) Publish(ctx context.Context, task *Task) error {
	if a.queue == nil {
		return nil // Если очередь не инициализирована, игнорируем
	}

	queueTask := &queue.Task{
		ID:         task.ID,
		Type:       task.Type,
		Data:       task.Data,
		ExecuteAt:  task.ExecuteAt,
		MaxRetries: task.MaxRetries,
		Attempts:   task.Attempts,
	}

	return a.queue.Publish(ctx, queueTask)
}
