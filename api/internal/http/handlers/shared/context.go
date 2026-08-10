package shared

import (
	"github.com/dujiao-next/internal/http/response"

	"github.com/gin-gonic/gin"
)

// GetAdminID ????????? ID?
func GetAdminID(c *gin.Context) (uint, bool) {
	return GetContextUintWithKeys(c, "admin_id", "error.admin_id_invalid", "error.admin_id_type_invalid")
}

// GetUserID ???????? ID?
func GetUserID(c *gin.Context) (uint, bool) {
	return GetContextUintWithKeys(c, "user_id", "error.user_id_invalid", "error.user_id_type_invalid")
}

// GetContextUintWithKeys ?????? uint ???????????
func GetContextUintWithKeys(c *gin.Context, key, invalidKey, typeInvalidKey string) (uint, bool) {
	value, exists := c.Get(key)
	if !exists {
		RespondError(c, response.CodeUnauthorized, "error.unauthorized", nil)
		return 0, false
	}

	switch v := value.(type) {
	case uint:
		return v, true
	case int:
		if v < 0 {
			RespondError(c, response.CodeBadRequest, invalidKey, nil)
			return 0, false
		}
		return uint(v), true
	case float64:
		if v < 0 {
			RespondError(c, response.CodeBadRequest, invalidKey, nil)
			return 0, false
		}
		return uint(v), true
	default:
		RespondError(c, response.CodeInternal, typeInvalidKey, nil)
		return 0, false
	}
}
