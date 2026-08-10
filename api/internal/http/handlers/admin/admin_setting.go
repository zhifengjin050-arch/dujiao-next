package admin

import (
	"github.com/dujiao-next/internal/cache"
	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/http/handlers/shared"
	"github.com/dujiao-next/internal/http/response"

	"github.com/gin-gonic/gin"
)

// ====================  ????  ====================

// GetSettings ????
func (h *Handler) GetSettings(c *gin.Context) {
	key := c.DefaultQuery("key", constants.SettingKeySiteConfig)

	value, err := h.SettingService.GetByKey(key)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.settings_fetch_failed", err)
		return
	}
	if value == nil {
		response.Success(c, gin.H{})
		return
	}

	response.Success(c, value)
}

// UpdateSettingsRequest ??????
type UpdateSettingsRequest struct {
	Key   string                 `json:"key" binding:"required"`
	Value map[string]interface{} `json:"value" binding:"required"`
}

// UpdateSettings ????
func (h *Handler) UpdateSettings(c *gin.Context) {
	var req UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}

	value, err := h.SettingService.Update(req.Key, req.Value)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.settings_save_failed", err)
		return
	}

	if req.Key == constants.SettingKeySiteConfig {
		_ = cache.Del(c.Request.Context(), publicConfigCacheKey)
	}
	if req.Key == constants.SettingKeyRegistrationConfig {
		_ = cache.Del(c.Request.Context(), publicConfigCacheKey)
	}
	if req.Key == constants.SettingKeyNavConfig {
		_ = cache.Del(c.Request.Context(), publicConfigCacheKey)
	}
	if req.Key == constants.SettingKeyWalletConfig {
		_ = cache.Del(c.Request.Context(), publicConfigCacheKey)
	}
	if req.Key == constants.SettingKeyCallbackRoutesConfig {
		h.SettingService.InvalidateCallbackRoutesCache()
	}
	response.Success(c, value)
}
