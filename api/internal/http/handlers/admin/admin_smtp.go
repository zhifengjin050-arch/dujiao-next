package admin

import (
	"errors"
	"strings"

	"github.com/dujiao-next/internal/http/handlers/shared"
	"github.com/dujiao-next/internal/http/response"
	"github.com/dujiao-next/internal/service"

	"github.com/gin-gonic/gin"
)

// GetSMTPSettings ?? SMTP ??(??)
func (h *Handler) GetSMTPSettings(c *gin.Context) {
	setting, err := h.SettingService.GetSMTPSetting(h.Config.Email)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.settings_fetch_failed", err)
		return
	}
	response.Success(c, service.MaskSMTPSettingForAdmin(setting))
}

// UpdateSMTPSettings ?? SMTP ??
func (h *Handler) UpdateSMTPSettings(c *gin.Context) {
	var req service.SMTPSettingPatch
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}

	setting, err := h.SettingService.PatchSMTPSetting(h.Config.Email, req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrSMTPConfigInvalid):
			shared.RespondErrorWithMsg(c, response.CodeBadRequest, err.Error(), nil)
		default:
			shared.RespondError(c, response.CodeInternal, "error.settings_save_failed", err)
		}
		return
	}

	h.Config.Email = service.SMTPSettingToConfig(setting)
	if h.EmailService != nil {
		h.EmailService.SetConfig(&h.Config.Email)
	}

	response.Success(c, service.MaskSMTPSettingForAdmin(setting))
}

// SMTPTestSendRequest SMTP ??????
type SMTPTestSendRequest struct {
	ToEmail string `json:"to_email" binding:"required"`
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

// TestSMTPSettings ?? SMTP ????
func (h *Handler) TestSMTPSettings(c *gin.Context) {
	var req SMTPTestSendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}

	toEmail := strings.TrimSpace(req.ToEmail)
	if toEmail == "" {
		shared.RespondError(c, response.CodeBadRequest, "error.email_invalid", nil)
		return
	}

	setting, err := h.SettingService.GetSMTPSetting(h.Config.Email)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.settings_fetch_failed", err)
		return
	}

	configForSend := service.SMTPSettingToConfig(setting)
	configForSend.Enabled = true
	tempEmailService := service.NewEmailService(&configForSend)

	if err := tempEmailService.SendCustomEmail(toEmail, req.Subject, req.Body); err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidEmail):
			shared.RespondError(c, response.CodeBadRequest, "error.email_invalid", nil)
		case errors.Is(err, service.ErrEmailRecipientRejected):
			shared.RespondError(c, response.CodeBadRequest, "error.email_recipient_not_found", nil)
		case errors.Is(err, service.ErrEmailServiceDisabled),
			errors.Is(err, service.ErrEmailServiceNotConfigured):
			shared.RespondError(c, response.CodeBadRequest, "error.email_service_not_configured", err)
		default:
			shared.RespondError(c, response.CodeInternal, "error.send_verify_code_failed", err)
		}
		return
	}

	response.Success(c, gin.H{"sent": true})
}
