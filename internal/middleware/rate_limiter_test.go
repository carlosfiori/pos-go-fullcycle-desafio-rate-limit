package middleware

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/carlosfiori/pos-go-fullcycle-desafio-rate-limit/internal/limiter"
)

type mockStore struct {
	allowed bool
}

func (m *mockStore) Increment(ctx context.Context, key string, windowSec int) (int64, error) {
	if m.allowed {
		return 1, nil
	}
	return 11, nil
}

func (m *mockStore) IsBlocked(ctx context.Context, key string) (bool, error) {
	return !m.allowed, nil
}

func (m *mockStore) Block(ctx context.Context, key string, duration time.Duration) error {
	return nil
}

func TestRateLimiter_Middleware_Allowed(t *testing.T) {
	store := &mockStore{allowed: true}
	rl := limiter.NewRateLimiter(store, 10, 5*time.Minute, nil)

	handler := RateLimiter(rl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	body, _ := io.ReadAll(rec.Body)
	if string(body) != "success" {
		t.Errorf("expected body 'success', got %s", string(body))
	}
}

func TestRateLimiter_Middleware_Blocked(t *testing.T) {
	store := &mockStore{allowed: false}
	rl := limiter.NewRateLimiter(store, 10, 5*time.Minute, nil)

	handler := RateLimiter(rl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("success"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected status 429, got %d", rec.Code)
	}

	body, _ := io.ReadAll(rec.Body)
	expectedMsg := "you have reached the maximum number of requests or actions allowed within a certain time frame\n"
	if string(body) != expectedMsg {
		t.Errorf("expected body %q, got %q", expectedMsg, string(body))
	}
}

func TestRateLimiter_Middleware_ExtractsAPIKey(t *testing.T) {
	var capturedToken string
	store := &mockStore{allowed: true}

	wrapper := &testLimiterWrapper{
		store: store,
		tokenHook: func(token string) {
			capturedToken = token
		},
	}

	rl := limiter.NewRateLimiter(wrapper, 10, 5*time.Minute, map[string]limiter.TokenConfig{
		"test-token": {
			Limit:         100,
			BlockDuration: 10 * time.Minute,
		},
	})

	handler := RateLimiter(rl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("API_KEY", "test-token")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if capturedToken != "test-token" {
		t.Errorf("expected token 'test-token', got %q", capturedToken)
	}
}

type testLimiterWrapper struct {
	store     *mockStore
	tokenHook func(string)
}

func (w *testLimiterWrapper) Increment(ctx context.Context, key string, windowSec int) (int64, error) {
	if w.tokenHook != nil && len(key) > 6 && key[:6] == "token:" {
		w.tokenHook(key[6:])
	}
	return w.store.Increment(ctx, key, windowSec)
}

func (w *testLimiterWrapper) IsBlocked(ctx context.Context, key string) (bool, error) {
	return w.store.IsBlocked(ctx, key)
}

func (w *testLimiterWrapper) Block(ctx context.Context, key string, duration time.Duration) error {
	return w.store.Block(ctx, key, duration)
}

func TestRateLimiter_Middleware_DifferentHTTPMethods(t *testing.T) {
	methods := []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodPatch}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			store := &mockStore{allowed: true}
			rl := limiter.NewRateLimiter(store, 10, 5*time.Minute, nil)

			handler := RateLimiter(rl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(method))
			}))

			req := httptest.NewRequest(method, "/", nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", rec.Code)
			}

			body, _ := io.ReadAll(rec.Body)
			if string(body) != method {
				t.Errorf("expected body %q, got %q", method, string(body))
			}
		})
	}
}

func TestRateLimiter_Middleware_WithoutRemoteAddr(t *testing.T) {
	store := &mockStore{allowed: true}
	rl := limiter.NewRateLimiter(store, 10, 5*time.Minute, nil)

	handler := RateLimiter(rl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = ""
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestRateLimiter_Middleware_MalformedIP(t *testing.T) {
	store := &mockStore{allowed: true}
	rl := limiter.NewRateLimiter(store, 10, 5*time.Minute, nil)

	handler := RateLimiter(rl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "invalid-ip-format"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestRateLimiter_Middleware_EmptyAPIKey(t *testing.T) {
	var capturedToken string
	store := &mockStore{allowed: true}

	wrapper := &testLimiterWrapper{
		store: store,
		tokenHook: func(token string) {
			capturedToken = token
		},
	}

	rl := limiter.NewRateLimiter(wrapper, 10, 5*time.Minute, nil)

	handler := RateLimiter(rl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("API_KEY", "")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	if capturedToken != "" {
		t.Errorf("expected empty token, got %q", capturedToken)
	}
}

func TestRateLimiter_Middleware_CaseInsensitiveHeader(t *testing.T) {
	store := &mockStore{allowed: true}
	rl := limiter.NewRateLimiter(store, 10, 5*time.Minute, map[string]limiter.TokenConfig{
		"test123": {
			Limit:         100,
			BlockDuration: 10 * time.Minute,
		},
	})

	handler := RateLimiter(rl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("api_key", "test123")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestRateLimiter_Middleware_MultipleRequests(t *testing.T) {
	callCount := 0
	store := &mockStore{allowed: true}
	rl := limiter.NewRateLimiter(store, 10, 5*time.Minute, nil)

	handler := RateLimiter(rl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("request %d: expected status 200, got %d", i+1, rec.Code)
		}
	}

	if callCount != 5 {
		t.Errorf("expected handler to be called 5 times, got %d", callCount)
	}
}

func TestRateLimiter_Middleware_BlockedDoesNotCallHandler(t *testing.T) {
	handlerCalled := false
	store := &mockStore{allowed: false}
	rl := limiter.NewRateLimiter(store, 10, 5*time.Minute, nil)

	handler := RateLimiter(rl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected status 429, got %d", rec.Code)
	}

	if handlerCalled {
		t.Error("expected handler not to be called when blocked")
	}
}

func TestRateLimiter_Middleware_DifferentPaths(t *testing.T) {
	paths := []string{"/", "/api/users", "/api/products", "/health"}

	for _, path := range paths {
		t.Run(path, func(t *testing.T) {
			store := &mockStore{allowed: true}
			rl := limiter.NewRateLimiter(store, 10, 5*time.Minute, nil)

			handler := RateLimiter(rl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(r.URL.Path))
			}))

			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()

			handler.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("expected status 200, got %d", rec.Code)
			}

			body, _ := io.ReadAll(rec.Body)
			if string(body) != path {
				t.Errorf("expected body %q, got %q", path, string(body))
			}
		})
	}
}

func TestRateLimiter_Middleware_WithIPPort(t *testing.T) {
	store := &mockStore{allowed: true}
	wrapper := &testLimiterWrapper{
		store:     store,
		tokenHook: nil,
	}
	rl := limiter.NewRateLimiter(wrapper, 10, 5*time.Minute, nil)

	handler := RateLimiter(rl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "192.168.1.1:54321"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestRateLimiter_Middleware_IPv6(t *testing.T) {
	store := &mockStore{allowed: true}
	rl := limiter.NewRateLimiter(store, 10, 5*time.Minute, nil)

	handler := RateLimiter(rl)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "[2001:db8::1]:8080"
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}
