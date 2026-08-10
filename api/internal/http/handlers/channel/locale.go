package channel

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/dujiao-next/internal/models"
)

var htmlTagRe = regexp.MustCompile(`<[^>]*>`)

// resolveLocalizedJSON ? models.JSON (map[string]interface{}) ? locale ?????
// ?? locale ? defaultLocale ? ??????
func resolveLocalizedJSON(m models.JSON, locale, defaultLocale string) string {
	if len(m) == 0 {
		return ""
	}
	if v, ok := m[locale]; ok {
		if s := fmt.Sprintf("%v", v); s != "" && s != "<nil>" {
			return s
		}
	}
	if v, ok := m[defaultLocale]; ok {
		if s := fmt.Sprintf("%v", v); s != "" && s != "<nil>" {
			return s
		}
	}
	for _, v := range m {
		if s := fmt.Sprintf("%v", v); s != "" && s != "<nil>" {
			return s
		}
	}
	return ""
}

// stripHTML ?? HTML ??,?????
func stripHTML(s string) string {
	text := htmlTagRe.ReplaceAllString(s, "")
	// ???????
	lines := strings.Split(text, "\n")
	var result []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return strings.Join(result, "\n")
}

// truncate ?????? n ? rune,????? "..."
func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}
