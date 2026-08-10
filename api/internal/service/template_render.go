package service

import (
	"fmt"
	"regexp"
	"strings"
)

var templateVarPattern = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_]+)\s*\}\}`)

// renderTemplate ?? {{variable}} ??
func renderTemplate(tmpl string, variables map[string]interface{}) string {
	tmpl = strings.TrimSpace(tmpl)
	if tmpl == "" {
		return ""
	}
	return templateVarPattern.ReplaceAllStringFunc(tmpl, func(matched string) string {
		submatch := templateVarPattern.FindStringSubmatch(matched)
		if len(submatch) != 2 {
			return matched
		}
		key := strings.TrimSpace(submatch[1])
		value, ok := variables[key]
		if !ok {
			return ""
		}
		return strings.TrimSpace(fmt.Sprintf("%v", value))
	})
}
