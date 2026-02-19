package limiter

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	client *redis.Client
}

func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{
		client: client,
	}
}

func (r *RedisStore) Increment(ctx context.Context, key string, windowSec int) (int64, error) {
	now := time.Now().Unix()
	windowKey := fmt.Sprintf("ratelimit:%s:%d", key, now)

	pipe := r.client.Pipeline()
	incrCmd := pipe.Incr(ctx, windowKey)
	pipe.Expire(ctx, windowKey, time.Duration(windowSec+1)*time.Second)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to increment rate limit: %w", err)
	}

	count := incrCmd.Val()
	return count, nil
}

func (r *RedisStore) IsBlocked(ctx context.Context, key string) (bool, error) {
	blockedKey := fmt.Sprintf("ratelimit:blocked:%s", key)
	exists, err := r.client.Exists(ctx, blockedKey).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check if blocked: %w", err)
	}
	return exists > 0, nil
}

func (r *RedisStore) Block(ctx context.Context, key string, duration time.Duration) error {
	blockedKey := fmt.Sprintf("ratelimit:blocked:%s", key)
	err := r.client.Set(ctx, blockedKey, 1, duration).Err()
	if err != nil {
		return fmt.Errorf("failed to block key: %w", err)
	}
	return nil
}
