package router

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/dujiao-next/internal/http/response"
	"github.com/dujiao-next/internal/i18n"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// RateLimitKeyFunc ???? key ???
type RateLimitKeyFunc func(*gin.Context) string

// RateLimitRule ????
type RateLimitRule struct {
	Prefix        string
	WindowSeconds int
	MaxRequests   int
	BlockSeconds  int
	MessageKey    string
}

var rateLimitScript = redis.NewScript(`
local current = redis.call("INCR", KEYS[1])
if current == 1 then
	redis.call("EXPIRE", KEYS[1], ARGV[1])
end
if tonumber(ARGV[2]) > 0 and tonumber(ARGV[3]) > 0 and current == tonumber(ARGV[2]) + 1 then
	redis.call("EXPIRE", KEYS[1], ARGV[3])
end
local ttl = redis.call("TTL", KEYS[1])
return {current, ttl}
`)

// RateLimitMiddleware Redis ???????
func RateLimitMiddleware(client *redis.Client, rule RateLimitRule, keyFunc RateLimitKeyFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if client == nil || rule.WindowSeconds <= 0 || rule.MaxRequests <= 0 {
			c.Next()
			return
		}

		key := ""
		if keyFunc != nil {
			key = strings.TrimSpace(keyFunc(c))
		}
		if key == "" {
			key = c.ClientIP()
		}
		if rule.Prefix != "" {
			key = fmt.Sprintf("%s:%s", rule.Prefix, key)
		}

		result, err := rateLimitScript.Run(
			c.Request.Context(),
			client,
			[]string{key},
			rule.WindowSeconds,
			rule.MaxRequests,
			rule.BlockSeconds,
		).Result()
		if err != nil {
			msg := i18n.T(i18n.ResolveLocale(c), "error.rate_limit_unavailable")
			if isChannelAPIRequest(c) {
				response.ChannelError(c, 500, response.CodeInternal, msg, "internal_error")
			} else {
				response.Error(c, response.CodeInternal, msg)
			}
			c.Abort()
			return
		}

		values, ok := result.([]interface{})
		if !ok || len(values) < 2 {
			msg := i18n.T(i18n.ResolveLocale(c), "error.rate_limit_unavailable")
			if isChannelAPIRequest(c) {
				response.ChannelError(c, 500, response.CodeInternal, msg, "internal_error")
			} else {
				response.Error(c, response.CodeInternal, msg)
			}
			c.Abort()
			return
		}
		count, ok := toInt64(values[0])
		if !ok {
			msg := i18n.T(i18n.ResolveLocale(c), "error.rate_limit_unavailable")
			if isChannelAPIRequest(c) {
				response.ChannelError(c, 500, response.CodeInternal, msg, "internal_error")
			} else {
				response.Error(c, response.CodeInternal, msg)
			}
			c.Abort()
			return
		}
		ttlSeconds, _ := toInt64(values[1])
		if count > int64(rule.MaxRequests) {
			waitSeconds := int(ttlSeconds)
			if waitSeconds < 1 {
				waitSeconds = rule.WindowSeconds
			}
			if waitSeconds < 1 {
				waitSeconds = 1
			}
			msgKey := strings.TrimSpace(rule.MessageKey)
			if msgKey == "" {
				msgKey = "error.rate_limited"
			}
			msg := i18n.Sprintf(i18n.ResolveLocale(c), msgKey, waitSeconds)
			if isChannelAPIRequest(c) {
				response.ChannelError(c, 429, response.CodeTooManyRequests, msg, "rate_limit_exceeded")
			} else {
				response.Error(c, response.CodeTooManyRequests, msg)
			}
			c.Abort()
			return
		}

		c.Next()
	}
}

func isChannelAPIRequest(c *gin.Context) bool {
	if c == nil || c.Request == nil {
		return false
	}
	return strings.HasPrefix(c.Request.URL.Path, "/api/v1/channel")
}

// KeyByIP ?? IP ???? key
func KeyByIP(c *gin.Context) string {
	return c.ClientIP()
}

// KeyByUpstreamApiKey ???? API Key ???? key
func KeyByUpstreamApiKey(c *gin.Context) string {
	apiKey := c.GetHeader("Dujiao-Next-Api-Key")
	if apiKey != "" {
		return apiKey
	}
	return c.ClientIP()
}

// KeyByIPAndJSONField ?? IP + JSON ?????? key
func KeyByIPAndJSONField(field string) RateLimitKeyFunc {
	return func(c *gin.Context) string {
		value := strings.ToLower(strings.TrimSpace(readJSONField(c, field)))
		if value == "" {
			return c.ClientIP()
		}
		return fmt.Sprintf("%s|%s", value, c.ClientIP())
	}
}

func readJSONField(c *gin.Context, field string) string {
	if c == nil || c.Request == nil || c.Request.Body == nil {
		return ""
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return ""
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	if len(body) == 0 {
		return ""
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	value, ok := payload[field]
	if !ok {
		return ""
	}
	if text, ok := value.(string); ok {
		return strings.TrimSpace(text)
	}
	return ""
}

func toInt64(value interface{}) (int64, bool) {
	switch v := value.(type) {
	case int64:
		return v, true
	case int:
		return int64(v), true
	case int32:
		return int64(v), true
	case int16:
		return int64(v), true
	case int8:
		return int64(v), true
	case uint64:
		return int64(v), true
	case uint32:
		return int64(v), true
	case uint16:
		return int64(v), true
	case uint8:
		return int64(v), true
	case float64:
		return int64(v), true
	case float32:
		return int64(v), true
	default:
		return 0, false
	}
}
