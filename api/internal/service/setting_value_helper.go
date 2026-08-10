package service

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/dujiao-next/internal/constants"
)

// parseSettingInt ??????????
func parseSettingInt(value interface{}) (int, error) {
	switch v := value.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return int(i), nil
		}
		if f, err := v.Float64(); err == nil {
			return int(f), nil
		}
		return 0, fmt.Errorf("invalid json number")
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0, fmt.Errorf("empty string")
		}
		parsed, err := strconv.Atoi(trimmed)
		if err != nil {
			return 0, err
		}
		return parsed, nil
	default:
		return 0, fmt.Errorf("unsupported value type")
	}
}

// normalizeSiteCurrency ??????????
func normalizeSiteCurrency(raw interface{}) string {
	currency := strings.ToUpper(normalizeSettingText(raw))
	if !settingCurrencyCodePattern.MatchString(currency) {
		return constants.SiteCurrencyDefault
	}
	return currency
}

// parseSettingFloat ??????????
func parseSettingFloat(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float32:
		return float64(v), nil
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case json.Number:
		return v.Float64()
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return 0, fmt.Errorf("empty string")
		}
		return strconv.ParseFloat(trimmed, 64)
	default:
		return 0, fmt.Errorf("unsupported value type")
	}
}
