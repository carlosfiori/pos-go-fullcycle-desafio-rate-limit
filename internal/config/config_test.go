package config

import (
	"os"
	"testing"
	"time"
)

func TestLoad_DefaultValues(t *testing.T) {
	os.Clearenv()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.RedisAddr != "redis:6379" {
		t.Errorf("expected default RedisAddr 'redis:6379', got %s", cfg.RedisAddr)
	}

	if cfg.RedisPassword != "" {
		t.Errorf("expected empty default RedisPassword, got %s", cfg.RedisPassword)
	}

	if cfg.IPLimit != 10 {
		t.Errorf("expected default IPLimit 10, got %d", cfg.IPLimit)
	}

	if cfg.IPBlockDuration != 300*time.Second {
		t.Errorf("expected default IPBlockDuration 300s, got %v", cfg.IPBlockDuration)
	}

	if len(cfg.TokenConfigs) != 0 {
		t.Errorf("expected empty TokenConfigs, got %d items", len(cfg.TokenConfigs))
	}
}

func TestLoad_CustomValues(t *testing.T) {

	os.Clearenv()
	os.Setenv("REDIS_ADDR", "localhost:6380")
	os.Setenv("REDIS_PASSWORD", "secret")
	os.Setenv("RATE_LIMIT_IP", "50")
	os.Setenv("RATE_LIMIT_IP_BLOCK_DURATION", "600")
	os.Setenv("RATE_LIMIT_TOKENS", "token1:100:300")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.RedisAddr != "localhost:6380" {
		t.Errorf("expected RedisAddr 'localhost:6380', got %s", cfg.RedisAddr)
	}

	if cfg.RedisPassword != "secret" {
		t.Errorf("expected RedisPassword 'secret', got %s", cfg.RedisPassword)
	}

	if cfg.IPLimit != 50 {
		t.Errorf("expected IPLimit 50, got %d", cfg.IPLimit)
	}

	if cfg.IPBlockDuration != 600*time.Second {
		t.Errorf("expected IPBlockDuration 600s, got %v", cfg.IPBlockDuration)
	}

	if len(cfg.TokenConfigs) != 1 {
		t.Fatalf("expected 1 token config, got %d", len(cfg.TokenConfigs))
	}

	token, exists := cfg.TokenConfigs["token1"]
	if !exists {
		t.Fatal("expected token1 to exist")
	}

	if token.Limit != 100 {
		t.Errorf("expected token1 limit 100, got %d", token.Limit)
	}

	if token.BlockDuration != 300*time.Second {
		t.Errorf("expected token1 block duration 300s, got %v", token.BlockDuration)
	}
}

func TestLoad_InvalidIPLimit(t *testing.T) {
	os.Clearenv()
	os.Setenv("RATE_LIMIT_IP", "invalid")

	_, err := Load()
	if err == nil {
		t.Error("expected error for invalid RATE_LIMIT_IP")
	}
}

func TestLoad_InvalidBlockDuration(t *testing.T) {
	os.Clearenv()
	os.Setenv("RATE_LIMIT_IP_BLOCK_DURATION", "invalid")

	_, err := Load()
	if err == nil {
		t.Error("expected error for invalid RATE_LIMIT_IP_BLOCK_DURATION")
	}
}

func TestLoad_MultipleTokens(t *testing.T) {
	os.Clearenv()
	os.Setenv("RATE_LIMIT_TOKENS", "token1:100:300,token2:50:600,token3:200:120")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.TokenConfigs) != 3 {
		t.Fatalf("expected 3 token configs, got %d", len(cfg.TokenConfigs))
	}

	token1, exists := cfg.TokenConfigs["token1"]
	if !exists {
		t.Error("expected token1 to exist")
	}
	if token1.Limit != 100 {
		t.Errorf("expected token1 limit 100, got %d", token1.Limit)
	}
	if token1.BlockDuration != 300*time.Second {
		t.Errorf("expected token1 block duration 300s, got %v", token1.BlockDuration)
	}

	token2, exists := cfg.TokenConfigs["token2"]
	if !exists {
		t.Error("expected token2 to exist")
	}
	if token2.Limit != 50 {
		t.Errorf("expected token2 limit 50, got %d", token2.Limit)
	}
	if token2.BlockDuration != 600*time.Second {
		t.Errorf("expected token2 block duration 600s, got %v", token2.BlockDuration)
	}

	token3, exists := cfg.TokenConfigs["token3"]
	if !exists {
		t.Error("expected token3 to exist")
	}
	if token3.Limit != 200 {
		t.Errorf("expected token3 limit 200, got %d", token3.Limit)
	}
	if token3.BlockDuration != 120*time.Second {
		t.Errorf("expected token3 block duration 120s, got %v", token3.BlockDuration)
	}
}

func TestLoad_InvalidTokenFormat(t *testing.T) {
	tests := []struct {
		name   string
		tokens string
	}{
		{"missing parts", "token1:100"},
		{"invalid limit", "token1:invalid:300"},
		{"invalid block duration", "token1:100:invalid"},
		{"too many parts", "token1:100:300:extra"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Clearenv()
			os.Setenv("RATE_LIMIT_TOKENS", tt.tokens)

			_, err := Load()
			if err == nil {
				t.Errorf("expected error for tokens %q", tt.tokens)
			}
		})
	}
}

func TestLoad_EmptyTokenString(t *testing.T) {
	os.Clearenv()
	os.Setenv("RATE_LIMIT_TOKENS", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.TokenConfigs) != 0 {
		t.Errorf("expected empty TokenConfigs for empty string, got %d items", len(cfg.TokenConfigs))
	}
}

func TestLoad_TokensWithSpaces(t *testing.T) {
	os.Clearenv()
	os.Setenv("RATE_LIMIT_TOKENS", " token1:100:300 , token2:50:600 ")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.TokenConfigs) != 2 {
		t.Fatalf("expected 2 token configs, got %d", len(cfg.TokenConfigs))
	}

	_, exists := cfg.TokenConfigs["token1"]
	if !exists {
		t.Error("expected token1 to exist after trimming spaces")
	}

	_, exists = cfg.TokenConfigs["token2"]
	if !exists {
		t.Error("expected token2 to exist after trimming spaces")
	}
}

func TestParseTokenConfigs_EmptyString(t *testing.T) {
	configs, err := parseTokenConfigs("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(configs) != 0 {
		t.Errorf("expected empty map, got %d items", len(configs))
	}
}

func TestParseTokenConfigs_SingleToken(t *testing.T) {
	configs, err := parseTokenConfigs("mytoken:75:450")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(configs) != 1 {
		t.Fatalf("expected 1 config, got %d", len(configs))
	}

	config, exists := configs["mytoken"]
	if !exists {
		t.Fatal("expected mytoken to exist")
	}

	if config.Limit != 75 {
		t.Errorf("expected limit 75, got %d", config.Limit)
	}

	if config.BlockDuration != 450*time.Second {
		t.Errorf("expected block duration 450s, got %v", config.BlockDuration)
	}
}

func TestGetEnv_ReturnsValue(t *testing.T) {
	os.Clearenv()
	os.Setenv("TEST_KEY", "test_value")

	value := getEnv("TEST_KEY", "default")
	if value != "test_value" {
		t.Errorf("expected 'test_value', got %s", value)
	}
}

func TestGetEnv_ReturnsDefault(t *testing.T) {
	os.Clearenv()

	value := getEnv("NONEXISTENT_KEY", "default_value")
	if value != "default_value" {
		t.Errorf("expected 'default_value', got %s", value)
	}
}

func TestLoad_NegativeValues(t *testing.T) {
	os.Clearenv()
	os.Setenv("RATE_LIMIT_IP", "-10")

	_, err := Load()

	if err != nil {
		t.Logf("Negative values cause error: %v", err)
	}
}

func TestLoad_ZeroValues(t *testing.T) {
	os.Clearenv()
	os.Setenv("RATE_LIMIT_IP", "0")
	os.Setenv("RATE_LIMIT_IP_BLOCK_DURATION", "0")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.IPLimit != 0 {
		t.Errorf("expected IPLimit 0, got %d", cfg.IPLimit)
	}

	if cfg.IPBlockDuration != 0 {
		t.Errorf("expected IPBlockDuration 0, got %v", cfg.IPBlockDuration)
	}
}
