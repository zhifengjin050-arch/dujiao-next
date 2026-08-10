package service

import "strings"

// normalizeSettingText ????????????
func normalizeSettingText(raw interface{}) string {
	text, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(text)
}

// normalizeSettingTextWithRuneLimit ?????????????
func normalizeSettingTextWithRuneLimit(raw interface{}, maxRuneCount int) string {
	text := normalizeSettingText(raw)
	if text == "" || maxRuneCount <= 0 {
		return text
	}

	runes := []rune(text)
	if len(runes) <= maxRuneCount {
		return text
	}
	return string(runes[:maxRuneCount])
}

// parseSettingBool ??????????
func parseSettingBool(raw interface{}) bool {
	switch value := raw.(type) {
	case bool:
		return value
	case int:
		return value != 0
	case int64:
		return value != 0
	case float64:
		return value != 0
	case string:
		normalized := strings.ToLower(strings.TrimSpace(value))
		return normalized == "1" || normalized == "true" || normalized == "yes" || normalized == "on"
	default:
		return false
	}
}

// cloneStringSlice ???????,?????????
func cloneStringSlice(items []string) []string {
	if len(items) == 0 {
		return []string{}
	}
	result := make([]string, len(items))
	copy(result, items)
	return result
}

// cloneUintSlice ?????????,?????????
func cloneUintSlice(items []uint) []uint {
	if len(items) == 0 {
		return []uint{}
	}
	result := make([]uint, len(items))
	copy(result, items)
	return result
}

// readStringList ? map ????????,???????????
func readStringList(source map[string]interface{}, key string, fallback []string) []string {
	value, ok := source[key]
	if !ok {
		return cloneStringSlice(fallback)
	}
	switch raw := value.(type) {
	case []string:
		return cloneStringSlice(raw)
	case []interface{}:
		result := make([]string, 0, len(raw))
		for _, item := range raw {
			if text, ok := item.(string); ok {
				result = append(result, text)
			}
		}
		return result
	default:
		return cloneStringSlice(fallback)
	}
}

// readUintList ? map ??????????,???????????
func readUintList(source map[string]interface{}, key string, fallback []uint) []uint {
	value, ok := source[key]
	if !ok {
		return cloneUintSlice(fallback)
	}
	switch raw := value.(type) {
	case []uint:
		return cloneUintSlice(raw)
	case []interface{}:
		result := make([]uint, 0, len(raw))
		for _, item := range raw {
			switch typed := item.(type) {
			case int:
				if typed > 0 {
					result = append(result, uint(typed))
				}
			case int64:
				if typed > 0 {
					result = append(result, uint(typed))
				}
			case uint:
				if typed > 0 {
					result = append(result, typed)
				}
			case float64:
				if typed > 0 {
					result = append(result, uint(typed))
				}
			}
		}
		return result
	default:
		return cloneUintSlice(fallback)
	}
}

// normalizeSettingStringList ??????????????
func normalizeSettingStringList(raw interface{}) []string {
	switch value := raw.(type) {
	case []string:
		return append([]string(nil), value...)
	case []interface{}:
		items := make([]string, 0, len(value))
		for _, item := range value {
			items = append(items, normalizeSettingText(item))
		}
		return items
	default:
		return nil
	}
}
