package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ds124wfegd/WB_L3/1/internal/database"
	"github.com/ds124wfegd/WB_L3/1/internal/entity"
	"github.com/ds124wfegd/WB_L3/1/internal/rabbitMQ"

	"github.com/google/uuid"
)

type notificationUseCase struct {
	repo        database.NotificationRepository
	queue       rabbitMQ.Queue
	maxAttempts int
}

func NewNotificationUseCase(repo database.NotificationRepository, q rabbitMQ.Queue, maxAttempts int) NotificationUseCase {
	return &notificationUseCase{
		repo:        repo,
		queue:       q,
		maxAttempts: maxAttempts,
	}
}

func (uc *notificationUseCase) CreateNotification(ctx context.Context, req *entity.NotificationRequest) (*entity.Notification, error) {
	notification := &entity.Notification{
		ID:        uuid.New().String(),
		UserID:    req.UserID,
		Title:     req.Title,
		Message:   req.Message,
		SendTime:  req.SendTime,
		Status:    entity.StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Attempts:  0,
	}

	if err := uc.repo.Create(ctx, notification); err != nil {
		return nil, err
	}

	// Schedule notification in queue with context
	delay := notification.SendTime.Sub(time.Now())
	if delay > 0 {
		if err := uc.queue.PublishWithDelay(ctx, notification, delay); err != nil {
			return nil, err
		}
	} else {
		// Если время уже настало, отправляем сразу
		if err := uc.queue.Publish(ctx, notification); err != nil {
			return nil, err
		}
	}

	return notification, nil
}

func (uc *notificationUseCase) GetNotification(ctx context.Context, id string) (*entity.Notification, error) {
	return uc.repo.GetByID(ctx, id)
}

func (uc *notificationUseCase) CancelNotification(ctx context.Context, id string) error {
	notification, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if notification == nil {
		return fmt.Errorf("notification not found")
	}

	notification.Status = entity.StatusCancelled
	notification.UpdatedAt = time.Now()

	return uc.repo.Update(ctx, notification)
}

func (uc *notificationUseCase) ProcessScheduledNotifications(ctx context.Context) error {
	pending, err := uc.repo.GetPendingNotifications(ctx)
	if err != nil {
		return err
	}

	now := time.Now()
	for _, notification := range pending {
		if notification.SendTime.Before(now) || notification.SendTime.Equal(now) {
			if err := uc.sendNotification(ctx, notification); err != nil {
				fmt.Printf("Failed to send notification %s: %v\n", notification.ID, err)
			}
		}
	}

	return nil
}

func (uc *notificationUseCase) sendNotification(ctx context.Context, notification *entity.Notification) error {
	// Симуляция отправки сообщений в <...>
	fmt.Printf("Sending notification to user %s: %s - %s\n",
		notification.UserID, notification.Title, notification.Message)

	// В будущем тут может быть реализация отправки сообщений в mail.ru
	notification.Status = entity.StatusSent
	notification.UpdatedAt = time.Now()

	return uc.repo.Update(ctx, notification)
}

func (s *notificationUseCase) GetAllNotifications(ctx context.Context) ([]*entity.Notification, error) {
	notifications, err := s.repo.GetAllNotifications(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get notifications from repository: %w", err)
	}
	return notifications, nil
}
