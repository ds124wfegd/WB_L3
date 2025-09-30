package service

import (
	"context"

	"github.com/ds124wfegd/WB_L3/1/internal/entity"
)

type NotificationUseCase interface {
	CreateNotification(ctx context.Context, req *entity.NotificationRequest) (*entity.Notification, error)
	GetNotification(ctx context.Context, id string) (*entity.Notification, error)
	CancelNotification(ctx context.Context, id string) error
	ProcessScheduledNotifications(ctx context.Context) error
	GetAllNotifications(ctx context.Context) ([]*entity.Notification, error)
}
