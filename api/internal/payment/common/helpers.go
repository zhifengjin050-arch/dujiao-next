package common

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// ReadString ? map ????????
func ReadString(raw map[string]interface{}, key string) string {
	if raw == nil || key == "" {
		return ""
	}
	value, ok := raw[key]
	if !ok || value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case json.Number:
		return v.String()
	case float64:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case int:
		return strconv.Itoa(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// ReadMap ? map ???? map?
func ReadMap(raw map[string]interface{}, key string) map[string]interface{} {
	if raw == nil || key == "" {
		return nil
	}
	value, ok := raw[key]
	if !ok || value == nil {
		return nil
	}
	if m, ok := value.(map[string]interface{}); ok {
		return m
	}
	return nil
}
