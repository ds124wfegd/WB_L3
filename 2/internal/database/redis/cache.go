package redis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ds124wfegd/WB_L3/2/internal/entity"

	"github.com/redis/go-redis/v9"
)

type CacheRepository struct {
	client *redis.Client
	ctx    context.Context
	ttl    time.Duration
}

func NewCacheRepository(client *redis.Client, ttl time.Duration) *CacheRepository {
	return &CacheRepository{
		client: client,
		ctx:    context.Background(),
		ttl:    ttl,
	}
}

func (r *CacheRepository) SetURL(shortURL string, url *entity.URL) error {
	data, err := json.Marshal(url)
	if err != nil {
		return err
	}

	return r.client.Set(r.ctx, "url:"+shortURL, data, r.ttl).Err()
}

func (r *CacheRepository) GetURL(shortURL string) (*entity.URL, error) {
	data, err := r.client.Get(r.ctx, "url:"+shortURL).Result()
	if err != nil {
		return nil, err
	}

	var url entity.URL
	err = json.Unmarshal([]byte(data), &url)
	if err != nil {
		return nil, err
	}

	return &url, nil
}

func (r *CacheRepository) DeleteURL(shortURL string) error {
	return r.client.Del(r.ctx, "url:"+shortURL).Err()
}

func (r *CacheRepository) IncrementPopularity(shortURL string) error {
	return r.client.ZIncrBy(r.ctx, "popular_urls", 1, shortURL).Err()
}

func (r *CacheRepository) GetPopularURLs(count int) ([]string, error) {
	result, err := r.client.ZRevRange(r.ctx, "popular_urls", 0, int64(count-1)).Result()
	if err != nil {
		return nil, err
	}
	return result, nil
}
