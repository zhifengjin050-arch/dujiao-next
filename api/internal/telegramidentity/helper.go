package telegramidentity

import (
	"fmt"
	"strings"
)

const (
	placeholderEmailPrefix = "telegram_"
	placeholderEmailDomain = "@login.local"
	defaultDisplayName     = "Telegram User"
)

// BuildPlaceholderEmail ?? Telegram ????????
func BuildPlaceholderEmail(providerUserID string) string {
	normalizedID := strings.TrimSpace(providerUserID)
	if normalizedID == "" {
		normalizedID = "unknown"
	}
	return fmt.Sprintf("%s%s%s", placeholderEmailPrefix, normalizedID, placeholderEmailDomain)
}

// IsPlaceholderEmail ????? Telegram ????????
func IsPlaceholderEmail(email string) bool {
	normalized := strings.ToLower(strings.TrimSpace(email))
	if normalized == "" {
		return false
	}
	return strings.HasPrefix(normalized, placeholderEmailPrefix) &&
		strings.HasSuffix(normalized, placeholderEmailDomain)
}

// ResolveDisplayName ?? Telegram ??????????
func ResolveDisplayName(providerUserID, username, firstName, lastName string) string {
	fullName := strings.TrimSpace(strings.TrimSpace(firstName) + " " + strings.TrimSpace(lastName))
	if fullName != "" {
		return fullName
	}
	if strings.TrimSpace(username) != "" {
		return strings.TrimSpace(username)
	}
	if strings.TrimSpace(providerUserID) != "" {
		return fmt.Sprintf("telegram_%s", strings.TrimSpace(providerUserID))
	}
	return defaultDisplayName
}
