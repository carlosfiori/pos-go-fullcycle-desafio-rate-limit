package limiter

import (
	"context"
	"errors"
	"testing"
	"time"
)

type mockStore struct {
	incrementFunc func(ctx context.Context, key string, windowSec int) (int64, error)
	isBlockedFunc func(ctx context.Context, key string) (bool, error)
	blockFunc     func(ctx context.Context, key string, duration time.Duration) error
}

func (m *mockStore) Increment(ctx context.Context, key string, windowSec int) (int64, error) {
	if m.incrementFunc != nil {
		return m.incrementFunc(ctx, key, windowSec)
	}
	return 0, nil
}

func (m *mockStore) IsBlocked(ctx context.Context, key string) (bool, error) {
	if m.isBlockedFunc != nil {
		return m.isBlockedFunc(ctx, key)
	}
	return false, nil
}

func (m *mockStore) Block(ctx context.Context, key string, duration time.Duration) error {
	if m.blockFunc != nil {
		return m.blockFunc(ctx, key, duration)
	}
	return nil
}

func TestRateLimiter_Allow_BelowLimit(t *testing.T) {
	store := &mockStore{
		incrementFunc: func(ctx context.Context, key string, windowSec int) (int64, error) {
			return 5, nil
		},
		isBlockedFunc: func(ctx context.Context, key string) (bool, error) {
			return false, nil
		},
	}

	rl := NewRateLimiter(store, 10, 5*time.Minute, nil)

	allowed, err := rl.Allow(context.Background(), "192.168.1.1", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Error("expected request to be allowed")
	}
}

func TestRateLimiter_Allow_ExceedsLimit(t *testing.T) {
	blockCalled := false
	store := &mockStore{
		incrementFunc: func(ctx context.Context, key string, windowSec int) (int64, error) {
			return 11, nil
		},
		isBlockedFunc: func(ctx context.Context, key string) (bool, error) {
			return false, nil
		},
		blockFunc: func(ctx context.Context, key string, duration time.Duration) error {
			blockCalled = true
			if duration != 5*time.Minute {
				t.Errorf("expected block duration 5m, got %v", duration)
			}
			return nil
		},
	}

	rl := NewRateLimiter(store, 10, 5*time.Minute, nil)

	allowed, err := rl.Allow(context.Background(), "192.168.1.1", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Error("expected request to be blocked")
	}
	if !blockCalled {
		t.Error("expected Block to be called")
	}
}

func TestRateLimiter_Allow_AlreadyBlocked(t *testing.T) {
	incrementCalled := false
	store := &mockStore{
		isBlockedFunc: func(ctx context.Context, key string) (bool, error) {
			return true, nil
		},
		incrementFunc: func(ctx context.Context, key string, windowSec int) (int64, error) {
			incrementCalled = true
			return 1, nil
		},
	}

	rl := NewRateLimiter(store, 10, 5*time.Minute, nil)

	allowed, err := rl.Allow(context.Background(), "192.168.1.1", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Error("expected request to be blocked")
	}
	if incrementCalled {
		t.Error("expected Increment not to be called when already blocked")
	}
}

func TestRateLimiter_Allow_TokenConfigTakesPrecedence(t *testing.T) {
	var capturedKey string
	store := &mockStore{
		incrementFunc: func(ctx context.Context, key string, windowSec int) (int64, error) {
			capturedKey = key
			return 50, nil
		},
		isBlockedFunc: func(ctx context.Context, key string) (bool, error) {
			return false, nil
		},
	}

	tokenConfigs := map[string]TokenConfig{
		"abc123": {
			Limit:         100,
			BlockDuration: 10 * time.Minute,
		},
	}

	rl := NewRateLimiter(store, 10, 5*time.Minute, tokenConfigs)

	allowed, err := rl.Allow(context.Background(), "192.168.1.1", "abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Error("expected request to be allowed with token config")
	}
	if capturedKey != "token:abc123" {
		t.Errorf("expected key token:abc123, got %s", capturedKey)
	}
}

func TestRateLimiter_Allow_UnconfiguredTokenUsesIPLimit(t *testing.T) {
	var capturedKey string
	store := &mockStore{
		incrementFunc: func(ctx context.Context, key string, windowSec int) (int64, error) {
			capturedKey = key
			return 5, nil
		},
		isBlockedFunc: func(ctx context.Context, key string) (bool, error) {
			return false, nil
		},
	}

	tokenConfigs := map[string]TokenConfig{
		"abc123": {
			Limit:         100,
			BlockDuration: 10 * time.Minute,
		},
	}

	rl := NewRateLimiter(store, 10, 5*time.Minute, tokenConfigs)

	allowed, err := rl.Allow(context.Background(), "192.168.1.1", "unknown_token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Error("expected request to be allowed")
	}
	if capturedKey != "ip:192.168.1.1" {
		t.Errorf("expected key ip:192.168.1.1, got %s", capturedKey)
	}
}

func TestRateLimiter_Allow_StoreError(t *testing.T) {
	testErr := errors.New("store error")
	store := &mockStore{
		isBlockedFunc: func(ctx context.Context, key string) (bool, error) {
			return false, testErr
		},
	}

	rl := NewRateLimiter(store, 10, 5*time.Minute, nil)

	allowed, err := rl.Allow(context.Background(), "192.168.1.1", "")
	if err != testErr {
		t.Errorf("expected error %v, got %v", testErr, err)
	}
	if allowed {
		t.Error("expected request to be blocked on error")
	}
}

func TestRateLimiter_Allow_ExactLimit(t *testing.T) {
	store := &mockStore{
		incrementFunc: func(ctx context.Context, key string, windowSec int) (int64, error) {
			return 10, nil
		},
		isBlockedFunc: func(ctx context.Context, key string) (bool, error) {
			return false, nil
		},
	}

	rl := NewRateLimiter(store, 10, 5*time.Minute, nil)

	allowed, err := rl.Allow(context.Background(), "192.168.1.1", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Error("expected request to be allowed at exact limit")
	}
}

func TestRateLimiter_Allow_IncrementError(t *testing.T) {
	testErr := errors.New("increment error")
	store := &mockStore{
		incrementFunc: func(ctx context.Context, key string, windowSec int) (int64, error) {
			return 0, testErr
		},
		isBlockedFunc: func(ctx context.Context, key string) (bool, error) {
			return false, nil
		},
	}

	rl := NewRateLimiter(store, 10, 5*time.Minute, nil)

	allowed, err := rl.Allow(context.Background(), "192.168.1.1", "")
	if err != testErr {
		t.Errorf("expected error %v, got %v", testErr, err)
	}
	if allowed {
		t.Error("expected request to be blocked on increment error")
	}
}

func TestRateLimiter_Allow_BlockError(t *testing.T) {
	testErr := errors.New("block error")
	store := &mockStore{
		incrementFunc: func(ctx context.Context, key string, windowSec int) (int64, error) {
			return 11, nil
		},
		isBlockedFunc: func(ctx context.Context, key string) (bool, error) {
			return false, nil
		},
		blockFunc: func(ctx context.Context, key string, duration time.Duration) error {
			return testErr
		},
	}

	rl := NewRateLimiter(store, 10, 5*time.Minute, nil)

	allowed, err := rl.Allow(context.Background(), "192.168.1.1", "")
	if err != testErr {
		t.Errorf("expected error %v, got %v", testErr, err)
	}
	if allowed {
		t.Error("expected request to be blocked on block error")
	}
}

func TestRateLimiter_Allow_MultipleTokens(t *testing.T) {
	tests := []struct {
		name          string
		token         string
		expectedKey   string
		expectedLimit int
	}{
		{
			name:          "First token",
			token:         "token1",
			expectedKey:   "token:token1",
			expectedLimit: 100,
		},
		{
			name:          "Second token",
			token:         "token2",
			expectedKey:   "token:token2",
			expectedLimit: 50,
		},
		{
			name:          "Third token",
			token:         "token3",
			expectedKey:   "token:token3",
			expectedLimit: 200,
		},
	}

	tokenConfigs := map[string]TokenConfig{
		"token1": {Limit: 100, BlockDuration: 5 * time.Minute},
		"token2": {Limit: 50, BlockDuration: 10 * time.Minute},
		"token3": {Limit: 200, BlockDuration: 3 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedKey string
			store := &mockStore{
				incrementFunc: func(ctx context.Context, key string, windowSec int) (int64, error) {
					capturedKey = key
					return 1, nil
				},
				isBlockedFunc: func(ctx context.Context, key string) (bool, error) {
					return false, nil
				},
			}

			rl := NewRateLimiter(store, 10, 5*time.Minute, tokenConfigs)

			allowed, err := rl.Allow(context.Background(), "192.168.1.1", tt.token)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !allowed {
				t.Error("expected request to be allowed")
			}
			if capturedKey != tt.expectedKey {
				t.Errorf("expected key %s, got %s", tt.expectedKey, capturedKey)
			}
		})
	}
}

func TestRateLimiter_Allow_EmptyToken(t *testing.T) {
	var capturedKey string
	store := &mockStore{
		incrementFunc: func(ctx context.Context, key string, windowSec int) (int64, error) {
			capturedKey = key
			return 1, nil
		},
		isBlockedFunc: func(ctx context.Context, key string) (bool, error) {
			return false, nil
		},
	}

	rl := NewRateLimiter(store, 10, 5*time.Minute, nil)

	allowed, err := rl.Allow(context.Background(), "192.168.1.1", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !allowed {
		t.Error("expected request to be allowed")
	}
	if capturedKey != "ip:192.168.1.1" {
		t.Errorf("expected key ip:192.168.1.1, got %s", capturedKey)
	}
}

func TestRateLimiter_Allow_DifferentIPs(t *testing.T) {
	callCount := make(map[string]int)
	store := &mockStore{
		incrementFunc: func(ctx context.Context, key string, windowSec int) (int64, error) {
			callCount[key]++
			return int64(callCount[key]), nil
		},
		isBlockedFunc: func(ctx context.Context, key string) (bool, error) {
			return false, nil
		},
	}

	rl := NewRateLimiter(store, 10, 5*time.Minute, nil)

	ips := []string{"192.168.1.1", "192.168.1.2", "10.0.0.1"}
	for _, ip := range ips {
		allowed, err := rl.Allow(context.Background(), ip, "")
		if err != nil {
			t.Fatalf("unexpected error for IP %s: %v", ip, err)
		}
		if !allowed {
			t.Errorf("expected request to be allowed for IP %s", ip)
		}
	}

	if len(callCount) != 3 {
		t.Errorf("expected 3 different keys, got %d", len(callCount))
	}
}

func TestRateLimiter_Allow_ContextCanceled(t *testing.T) {
	store := &mockStore{
		isBlockedFunc: func(ctx context.Context, key string) (bool, error) {
			return false, ctx.Err()
		},
	}

	rl := NewRateLimiter(store, 10, 5*time.Minute, nil)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	allowed, err := rl.Allow(ctx, "192.168.1.1", "")
	if err != context.Canceled {
		t.Errorf("expected context.Canceled error, got %v", err)
	}
	if allowed {
		t.Error("expected request to be blocked on context error")
	}
}

func TestRateLimiter_Allow_TokenBlockDuration(t *testing.T) {
	var capturedDuration time.Duration
	store := &mockStore{
		incrementFunc: func(ctx context.Context, key string, windowSec int) (int64, error) {
			return 101, nil
		},
		isBlockedFunc: func(ctx context.Context, key string) (bool, error) {
			return false, nil
		},
		blockFunc: func(ctx context.Context, key string, duration time.Duration) error {
			capturedDuration = duration
			return nil
		},
	}

	tokenConfigs := map[string]TokenConfig{
		"special-token": {
			Limit:         100,
			BlockDuration: 15 * time.Minute,
		},
	}

	rl := NewRateLimiter(store, 10, 5*time.Minute, tokenConfigs)

	allowed, err := rl.Allow(context.Background(), "192.168.1.1", "special-token")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if allowed {
		t.Error("expected request to be blocked")
	}
	if capturedDuration != 15*time.Minute {
		t.Errorf("expected block duration 15m, got %v", capturedDuration)
	}
}
