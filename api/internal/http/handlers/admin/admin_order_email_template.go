package admin

import (
	"errors"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/http/handlers/shared"
	"github.com/dujiao-next/internal/http/response"
	"github.com/dujiao-next/internal/service"

	"github.com/gin-gonic/gin"
)

// GetOrderEmailTemplateSettings ??????????
func (h *Handler) GetOrderEmailTemplateSettings(c *gin.Context) {
	setting, err := h.SettingService.GetOrderEmailTemplateSetting()
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.settings_fetch_failed", err)
		return
	}
	response.Success(c, service.MaskOrderEmailTemplateSettingForAdmin(setting))
}

// UpdateOrderEmailTemplateSettings ??????????
func (h *Handler) UpdateOrderEmailTemplateSettings(c *gin.Context) {
	var req service.OrderEmailTemplateSettingPatch
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}

	setting, err := h.SettingService.PatchOrderEmailTemplateSetting(req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrOrderEmailTemplateConfigInvalid):
			shared.RespondErrorWithMsg(c, response.CodeBadRequest, err.Error(), nil)
		default:
			shared.RespondError(c, response.CodeInternal, "error.settings_save_failed", err)
		}
		return
	}

	response.Success(c, service.MaskOrderEmailTemplateSettingForAdmin(setting))
}

// ResetOrderEmailTemplateSettings ???????????
func (h *Handler) ResetOrderEmailTemplateSettings(c *gin.Context) {
	defaultSetting := service.OrderEmailTemplateDefaultSetting()
	if _, err := h.SettingService.Update(
		constants.SettingKeyOrderEmailTemplateConfig,
		service.OrderEmailTemplateSettingToMap(defaultSetting),
	); err != nil {
		shared.RespondError(c, response.CodeInternal, "error.settings_save_failed", err)
		return
	}
	response.Success(c, service.MaskOrderEmailTemplateSettingForAdmin(defaultSetting))
}
