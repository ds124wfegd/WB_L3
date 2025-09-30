package database

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ds124wfegd/WB_L3/1/internal/entity"

	"github.com/go-redis/redis/v8"
)

type redisRepository struct {
	client *redis.Client
}

func NewRedisRepository(client *redis.Client) NotificationRepository {
	return &redisRepository{client: client}
}

func (r *redisRepository) Create(ctx context.Context, notification *entity.Notification) error {
	data, err := json.Marshal(notification)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("notification:%s", notification.ID)
	return r.client.Set(ctx, key, data, 0).Err()
}

func (r *redisRepository) GetByID(ctx context.Context, id string) (*entity.Notification, error) {
	key := fmt.Sprintf("notification:%s", id)
	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	var notification entity.Notification
	err = json.Unmarshal([]byte(data), &notification)
	return &notification, err
}

func (r *redisRepository) Update(ctx context.Context, notification *entity.Notification) error {
	return r.Create(ctx, notification)
}

func (r *redisRepository) Delete(ctx context.Context, id string) error {
	key := fmt.Sprintf("notification:%s", id)
	return r.client.Del(ctx, key).Err()
}

func (r *redisRepository) GetPendingNotifications(ctx context.Context) ([]*entity.Notification, error) {
	keys, err := r.client.Keys(ctx, "notification:*").Result()
	if err != nil {
		return nil, err
	}

	var notifications []*entity.Notification
	for _, key := range keys {
		data, err := r.client.Get(ctx, key).Result()
		if err != nil {
			continue
		}

		var notification entity.Notification
		if err := json.Unmarshal([]byte(data), &notification); err != nil {
			continue
		}

		if notification.Status == entity.StatusPending {
			notifications = append(notifications, &notification)
		}
	}

	return notifications, nil
}

func (r *redisRepository) GetAllNotifications(ctx context.Context) ([]*entity.Notification, error) {
	keys, err := r.client.Keys(ctx, "notification:*").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get notification keys: %w", err)
	}

	var notifications []*entity.Notification

	for _, key := range keys {
		data, err := r.client.Get(ctx, key).Result()
		if err != nil {
			if err == redis.Nil {
				continue
			}
			return nil, fmt.Errorf("failed to get notification %s: %w", key, err)
		}

		var notification entity.Notification
		if err := json.Unmarshal([]byte(data), &notification); err != nil {
			fmt.Printf("Failed to unmarshal notification from key %s: %v\n", key, err)
			continue
		}

		notifications = append(notifications, &notification)
	}

	return notifications, nil
}
