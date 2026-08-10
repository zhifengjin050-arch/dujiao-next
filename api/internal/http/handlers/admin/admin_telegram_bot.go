package admin

import (
	"github.com/dujiao-next/internal/http/handlers/shared"
	"github.com/dujiao-next/internal/http/response"
	"github.com/dujiao-next/internal/service"

	"github.com/gin-gonic/gin"
)

// GetTelegramBotConfig ?? Telegram Bot ??
func (h *Handler) GetTelegramBotConfig(c *gin.Context) {
	setting, err := h.SettingService.GetTelegramBotConfig()
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.settings_fetch_failed", err)
		return
	}
	response.Success(c, service.MaskTelegramBotConfigForAdmin(*setting))
}

// UpdateTelegramBotConfig ?? Telegram Bot ??(?????)
func (h *Handler) UpdateTelegramBotConfig(c *gin.Context) {
	var req service.TelegramBotConfigSetting
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}

	setting, err := h.SettingService.UpdateTelegramBotConfig(req)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.settings_save_failed", err)
		return
	}

	response.Success(c, service.MaskTelegramBotConfigForAdmin(*setting))
}

// GetTelegramBotRuntimeStatus ?? Telegram Bot ?????
func (h *Handler) GetTelegramBotRuntimeStatus(c *gin.Context) {
	status, err := h.SettingService.GetTelegramBotRuntimeStatus()
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.settings_fetch_failed", err)
		return
	}
	response.Success(c, status)
}
