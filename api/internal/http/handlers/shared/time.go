package shared

import "time"

// ParseTimeNullable ????? RFC3339 ?????,???? nil?
func ParseTimeNullable(raw string) (*time.Time, error) {
	if raw == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}
