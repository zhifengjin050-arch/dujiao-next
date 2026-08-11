package cache

import (
	"context"
	"testing"
	"time"

	"github.com/dujiao-next/internal/config"
	"github.com/dujiao-next/internal/constants"
)

func TestInitRedisNilConfig(t *testing.T) {
	// 传入 nil 配置应该安全返回，不启用 Redis
	err := InitRedis(nil)
	if err != nil {
		t.Fatalf("InitRedis with nil config should not error, got: %v", err)
	}
	if Enabled() {
		t.Fatal("expected cache to be disabled with nil config")
	}
}

func TestInitRedisDisabled(t *testing.T) {
	// 传入 Enabled=false 的配置应该不启用 Redis
	cfg := &config.RedisConfig{
		Enabled: false,
		Host:    "127.0.0.1",
		Port:    6379,
	}
	err := InitRedis(cfg)
	if err != nil {
		t.Fatalf("InitRedis with disabled config should not error, got: %v", err)
	}
	if Enabled() {
		t.Fatal("expected cache to be disabled")
	}
}

func TestInitRedisEmptyHostUsesDefault(t *testing.T) {
	// 空的 Host 应该使用默认值 127.0.0.1
	cfg := &config.RedisConfig{
		Enabled: true,
		Host:    "",
		Port:    0,
		Prefix:  "",
	}
	err := InitRedis(cfg)
	if err != nil {
		t.Fatalf("InitRedis should not error, got: %v", err)
	}
	if !Enabled() {
		t.Fatal("expected cache to be enabled")
	}
	// 清理：重置为禁用状态
	redisEnabled = false
	redisClient = nil
}

func TestInitRedisEmptyPrefixUsesDefault(t *testing.T) {
	cfg := &config.RedisConfig{
		Enabled: true,
		Host:    "127.0.0.1",
		Port:    6379,
		Prefix:  "",
	}
	err := InitRedis(cfg)
	if err != nil {
		t.Fatalf("InitRedis should not error, got: %v", err)
	}
	if redisPrefix != constants.RedisPrefixDefault {
		t.Fatalf("expected prefix %q, got %q", constants.RedisPrefixDefault, redisPrefix)
	}
	// 清理
	redisEnabled = false
	redisClient = nil
}

func TestEnabledReturnsFalseWhenNotInitialized(t *testing.T) {
	redisEnabled = false
	redisClient = nil
	if Enabled() {
		t.Fatal("expected Enabled()=false when not initialized")
	}
}

func TestClientReturnsNilWhenDisabled(t *testing.T) {
	redisEnabled = false
	redisClient = nil
	if Client() != nil {
		t.Fatal("expected Client()=nil when disabled")
	}
}

func TestBuildKey(t *testing.T) {
	redisPrefix = "dujiao"

	tests := []struct {
		name     string
		key      string
		expected string
	}{
		{"normal key", "user:123", "dujiao:user:123"},
		{"key with spaces", "  cart:456  ", "dujiao:cart:456"},
		{"empty key", "", "dujiao"},
		{"whitespace only", "   ", "dujiao"},
		{"nested key", "auth:admin:1", "dujiao:auth:admin:1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildKey(tt.key)
			if result != tt.expected {
				t.Fatalf("buildKey(%q) = %q, want %q", tt.key, result, tt.expected)
			}
		})
	}
}

func TestBuildKeyEmptyPrefix(t *testing.T) {
	redisPrefix = ""
	result := buildKey("test")
	if result != ":test" {
		t.Fatalf("buildKey with empty prefix = %q, want :test", result)
	}
}

func TestGetJSONWhenDisabled(t *testing.T) {
	redisEnabled = false
	redisClient = nil

	var dest map[string]interface{}
	hit, err := GetJSON(context.Background(), "test", &dest)
	if err != nil {
		t.Fatalf("GetJSON when disabled should not error, got: %v", err)
	}
	if hit {
		t.Fatal("GetJSON when disabled should return hit=false")
	}
}

func TestSetJSONWhenDisabled(t *testing.T) {
	redisEnabled = false
	redisClient = nil

	err := SetJSON(context.Background(), "test", map[string]string{"key": "value"}, time.Minute)
	if err != nil {
		t.Fatalf("SetJSON when disabled should not error, got: %v", err)
	}
}

func TestGetStringWhenDisabled(t *testing.T) {
	redisEnabled = false
	redisClient = nil

	val, err := GetString(context.Background(), "test")
	if err != nil {
		t.Fatalf("GetString when disabled should not error, got: %v", err)
	}
	if val != "" {
		t.Fatalf("GetString when disabled should return empty string, got: %q", val)
	}
}

func TestSetStringWhenDisabled(t *testing.T) {
	redisEnabled = false
	redisClient = nil

	err := SetString(context.Background(), "test", "value", time.Minute)
	if err != nil {
		t.Fatalf("SetString when disabled should not error, got: %v", err)
	}
}

func TestDelWhenDisabled(t *testing.T) {
	redisEnabled = false
	redisClient = nil

	err := Del(context.Background(), "test")
	if err != nil {
		t.Fatalf("Del when disabled should not error, got: %v", err)
	}
}

func TestSetNXWhenDisabled(t *testing.T) {
	redisEnabled = false
	redisClient = nil

	ok, err := SetNX(context.Background(), "test", "value", time.Minute)
	if err != nil {
		t.Fatalf("SetNX when disabled should not error, got: %v", err)
	}
	if !ok {
		t.Fatal("SetNX when disabled should return ok=true")
	}
}
