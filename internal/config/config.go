package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/carlosfiori/pos-go-fullcycle-desafio-rate-limit/internal/limiter"
	"github.com/joho/godotenv"
)

type Config struct {
	RedisAddr       string
	RedisPassword   string
	IPLimit         int
	IPBlockDuration time.Duration
	TokenConfigs    map[string]limiter.TokenConfig
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		RedisAddr:     getEnv("REDIS_ADDR", "redis:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
	}

	ipLimit, err := strconv.Atoi(getEnv("RATE_LIMIT_IP", "10"))
	if err != nil {
		return nil, fmt.Errorf("invalid RATE_LIMIT_IP: %w", err)
	}
	cfg.IPLimit = ipLimit

	ipBlockSec, err := strconv.Atoi(getEnv("RATE_LIMIT_IP_BLOCK_DURATION", "300"))
	if err != nil {
		return nil, fmt.Errorf("invalid RATE_LIMIT_IP_BLOCK_DURATION: %w", err)
	}
	cfg.IPBlockDuration = time.Duration(ipBlockSec) * time.Second

	tokenConfigs, err := parseTokenConfigs(getEnv("RATE_LIMIT_TOKENS", ""))
	if err != nil {
		return nil, fmt.Errorf("invalid RATE_LIMIT_TOKENS: %w", err)
	}
	cfg.TokenConfigs = tokenConfigs

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func parseTokenConfigs(s string) (map[string]limiter.TokenConfig, error) {
	configs := make(map[string]limiter.TokenConfig)
	if s == "" {
		return configs, nil
	}

	entries := strings.Split(s, ",")
	for _, entry := range entries {
		parts := strings.Split(strings.TrimSpace(entry), ":")
		if len(parts) != 3 {
			return nil, fmt.Errorf("invalid token config format: %s (expected token:limit:blockSec)", entry)
		}

		token := strings.TrimSpace(parts[0])
		limit, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return nil, fmt.Errorf("invalid limit for token %s: %w", token, err)
		}

		blockSec, err := strconv.Atoi(strings.TrimSpace(parts[2]))
		if err != nil {
			return nil, fmt.Errorf("invalid block duration for token %s: %w", token, err)
		}

		configs[token] = limiter.TokenConfig{
			Limit:         limit,
			BlockDuration: time.Duration(blockSec) * time.Second,
		}
	}

	return configs, nil
}
