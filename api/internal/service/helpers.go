package service

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"net/url"
	"strings"
	"time"
)

// randNumeric ???????????????
func randNumeric(length int) string {
	if length <= 0 {
		return ""
	}
	var b strings.Builder
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			b.WriteString("0")
			continue
		}
		fmt.Fprintf(&b, "%d", n.Int64())
	}
	return b.String()
}

// generateSerialNo ?????????(?? + ??? + ????)?
func generateSerialNo(prefix string) string {
	now := time.Now().Format("20060102150405")
	return fmt.Sprintf("%s%s%s", prefix, now, randNumeric(6))
}

// pickFirstNonEmpty ???????(trim ?)?????
func pickFirstNonEmpty(values ...string) string {
	for _, val := range values {
		trimmed := strings.TrimSpace(val)
		if trimmed != "" {
			return trimmed
		}
	}
	return ""
}

// appendURLQuery ? URL ???????
func appendURLQuery(rawURL string, params map[string]string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	query := parsed.Query()
	for key, value := range params {
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" || value == "" {
			continue
		}
		query.Set(key, value)
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}
