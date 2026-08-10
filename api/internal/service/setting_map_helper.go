package service

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/dujiao-next/internal/models"
)

// toStringAnyMap ????????? map[string]interface{}?
func toStringAnyMap(value interface{}) map[string]interface{} {
	switch v := value.(type) {
	case map[string]interface{}:
		return v
	case models.JSON:
		result := make(map[string]interface{}, len(v))
		for key, item := range v {
			result[key] = item
		}
		return result
	default:
		return nil
	}
}

// readString ? map ????????,?????????
func readString(source map[string]interface{}, key, fallback string) string {
	value, ok := source[key]
	if !ok {
		return fallback
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	default:
		return fallback
	}
}

// readBool ? map ???????,?????????
func readBool(source map[string]interface{}, key string, fallback bool) bool {
	value, ok := source[key]
	if !ok {
		return fallback
	}
	switch v := value.(type) {
	case bool:
		return v
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "true", "1", "yes", "on":
			return true
		case "false", "0", "no", "off":
			return false
		default:
			return fallback
		}
	default:
		return fallback
	}
}

// readInt ? map ???????,?????????
func readInt(source map[string]interface{}, key string, fallback int) int {
	value, ok := source[key]
	if !ok {
		return fallback
	}
	switch v := value.(type) {
	case int:
		return v
	case int8:
		return int(v)
	case int16:
		return int(v)
	case int32:
		return int(v)
	case int64:
		return int(v)
	case uint:
		return int(v)
	case uint8:
		return int(v)
	case uint16:
		return int(v)
	case uint32:
		return int(v)
	case uint64:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return int(i)
		}
		if f, err := v.Float64(); err == nil {
			return int(f)
		}
		return fallback
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return fallback
		}
		parsed, err := strconv.Atoi(trimmed)
		if err != nil {
			return fallback
		}
		return parsed
	default:
		return fallback
	}
}
