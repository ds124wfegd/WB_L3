package database

import (
	"context"
	"time"

	"github.com/ds124wfegd/WB_L3/1/internal/entity"
)

type NotificationRepository interface {
	Create(ctx context.Context, notification *entity.Notification) error
	GetByID(ctx context.Context, id string) (*entity.Notification, error)
	Update(ctx context.Context, notification *entity.Notification) error
	Delete(ctx context.Context, id string) error
	GetPendingNotifications(ctx context.Context) ([]*entity.Notification, error)
	GetAllNotifications(ctx context.Context) ([]*entity.Notification, error)
}

type CacheRepository interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Delete(ctx context.Context, key string) error
}
