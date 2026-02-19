package limiter

import (
	"context"
	"time"
)

type Store interface {
	Increment(ctx context.Context, key string, windowSec int) (int64, error)

	IsBlocked(ctx context.Context, key string) (bool, error)

	Block(ctx context.Context, key string, duration time.Duration) error
}
