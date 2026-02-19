package main

import (
	"log"
	"net/http"

	"github.com/carlosfiori/pos-go-fullcycle-desafio-rate-limit/internal/config"
	"github.com/carlosfiori/pos-go-fullcycle-desafio-rate-limit/internal/limiter"
	"github.com/carlosfiori/pos-go-fullcycle-desafio-rate-limit/internal/middleware"
	"github.com/redis/go-redis/v9"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       0,
	})

	store := limiter.NewRedisStore(redisClient)

	rateLimiter := limiter.NewRateLimiter(
		store,
		cfg.IPLimit,
		cfg.IPBlockDuration,
		cfg.TokenConfigs,
	)

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK\n"))
	})

	handler := middleware.RateLimiter(rateLimiter)(mux)

	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
