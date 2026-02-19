package limiter

import (
	"context"
	"time"
)

type TokenConfig struct {
	Limit         int
	BlockDuration time.Duration
}

type RateLimiter struct {
	store           Store
	ipLimit         int
	ipBlockDuration time.Duration
	tokenConfigs    map[string]TokenConfig
}

func NewRateLimiter(store Store, ipLimit int, ipBlockDuration time.Duration, tokenConfigs map[string]TokenConfig) *RateLimiter {
	return &RateLimiter{
		store:           store,
		ipLimit:         ipLimit,
		ipBlockDuration: ipBlockDuration,
		tokenConfigs:    tokenConfigs,
	}
}

func (rl *RateLimiter) Allow(ctx context.Context, ip string, token string) (bool, error) {
	var key string
	var limit int
	var blockDuration time.Duration

	if token != "" {
		if config, exists := rl.tokenConfigs[token]; exists {
			key = "token:" + token
			limit = config.Limit
			blockDuration = config.BlockDuration
		} else {

			key = "ip:" + ip
			limit = rl.ipLimit
			blockDuration = rl.ipBlockDuration
		}
	} else {

		key = "ip:" + ip
		limit = rl.ipLimit
		blockDuration = rl.ipBlockDuration
	}

	blocked, err := rl.store.IsBlocked(ctx, key)
	if err != nil {
		return false, err
	}
	if blocked {
		return false, nil
	}

	count, err := rl.store.Increment(ctx, key, 1)
	if err != nil {
		return false, err
	}

	if count > int64(limit) {

		if err := rl.store.Block(ctx, key, blockDuration); err != nil {
			return false, err
		}
		return false, nil
	}

	return true, nil
}
