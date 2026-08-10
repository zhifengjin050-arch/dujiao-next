package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dujiao-next/internal/config"
	"github.com/dujiao-next/internal/constants"

	"github.com/redis/go-redis/v9"
)

var redisClient *redis.Client
var redisPrefix string
var redisEnabled bool

// InitRedis ??? Redis ???
func InitRedis(cfg *config.RedisConfig) error {
	if cfg == nil || !cfg.Enabled {
		redisEnabled = false
		return nil
	}
	addr := strings.TrimSpace(cfg.Host)
	if addr == "" {
		addr = "127.0.0.1"
	}
	port := cfg.Port
	if port <= 0 {
		port = 6379
	}
	redisPrefix = strings.TrimSpace(cfg.Prefix)
	if redisPrefix == "" {
		redisPrefix = constants.RedisPrefixDefault
	}

	redisClient = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", addr, port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	redisEnabled = true
	return nil
}

// Enabled ????????
func Enabled() bool {
	return redisEnabled && redisClient != nil
}

// Client ?? Redis ???
func Client() *redis.Client {
	if !Enabled() {
		return nil
	}
	return redisClient
}

// GetJSON ?? JSON ??
func GetJSON(ctx context.Context, key string, dest interface{}) (bool, error) {
	if !Enabled() {
		return false, nil
	}
	val, err := redisClient.Get(ctx, buildKey(key)).Result()
	if err == redis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if err := json.Unmarshal([]byte(val), dest); err != nil {
		return false, err
	}
	return true, nil
}

// SetJSON ?? JSON ??
func SetJSON(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if !Enabled() {
		return nil
	}
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return redisClient.Set(ctx, buildKey(key), payload, ttl).Err()
}

// GetString ???????
func GetString(ctx context.Context, key string) (string, error) {
	if !Enabled() {
		return "", nil
	}
	val, err := redisClient.Get(ctx, buildKey(key)).Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
}

// SetString ???????
func SetString(ctx context.Context, key string, value string, ttl time.Duration) error {
	if !Enabled() {
		return nil
	}
	return redisClient.Set(ctx, buildKey(key), value, ttl).Err()
}

// Del ????
func Del(ctx context.Context, key string) error {
	if !Enabled() {
		return nil
	}
	return redisClient.Del(ctx, buildKey(key)).Err()
}

// SetNX ????(????????????)
func SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	if !Enabled() {
		return true, nil
	}
	result, err := redisClient.SetNX(ctx, buildKey(key), value, ttl).Result()
	if err != nil {
		return false, err
	}
	return result, nil
}

func buildKey(key string) string {
	trimmed := strings.TrimSpace(key)
	if trimmed == "" {
		return redisPrefix
	}
	return fmt.Sprintf("%s:%s", redisPrefix, trimmed)
}
