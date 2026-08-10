package shared

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

// ParseParamUint ??????????? ID?
func ParseParamUint(c *gin.Context, key string) (uint, error) {
	if c == nil {
		return 0, errors.New("context is nil")
	}
	raw := strings.TrimSpace(c.Param(key))
	if raw == "" {
		return 0, errors.New("param is empty")
	}
	parsed, err := strconv.ParseUint(raw, 10, 64)
	if err != nil || parsed == 0 {
		return 0, errors.New("param is invalid")
	}
	return uint(parsed), nil
}

// ParseQueryUint ????????????? ID?
func ParseQueryUint(raw string, zeroInvalid bool) (uint, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, nil
	}
	parsed, err := strconv.ParseUint(trimmed, 10, 64)
	if err != nil {
		return 0, err
	}
	if zeroInvalid && parsed == 0 {
		return 0, errors.New("query value is invalid")
	}
	return uint(parsed), nil
}
