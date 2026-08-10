package admin

import (
	"errors"
	"strconv"
	"strings"

	"github.com/dujiao-next/internal/http/handlers/shared"
	"github.com/dujiao-next/internal/http/response"
	"github.com/dujiao-next/internal/repository"
	"github.com/dujiao-next/internal/service"

	"github.com/gin-gonic/gin"
)

// GetNotificationCenterSettings ????????
func (h *Handler) GetNotificationCenterSettings(c *gin.Context) {
	setting, err := h.SettingService.GetNotificationCenterSetting()
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.settings_fetch_failed", err)
		return
	}
	response.Success(c, service.MaskNotificationCenterSettingForAdmin(setting))
}

// UpdateNotificationCenterSettings ????????
func (h *Handler) UpdateNotificationCenterSettings(c *gin.Context) {
	var req service.NotificationCenterSettingPatch
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}

	setting, err := h.SettingService.PatchNotificationCenterSetting(req)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNotificationConfigInvalid):
			shared.RespondErrorWithMsg(c, response.CodeBadRequest, err.Error(), nil)
		default:
			shared.RespondError(c, response.CodeInternal, "error.settings_save_failed", err)
		}
		return
	}
	response.Success(c, service.MaskNotificationCenterSettingForAdmin(setting))
}

// NotificationCenterTestSendRequest ??????????
type NotificationCenterTestSendRequest struct {
	Channel   string                 `json:"channel" binding:"required"`
	Target    string                 `json:"target" binding:"required"`
	Scene     string                 `json:"scene"`
	Locale    string                 `json:"locale"`
	Variables map[string]interface{} `json:"variables"`
}

// ListNotificationLogs ??????????
func (h *Handler) ListNotificationLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	page, pageSize = shared.NormalizePagination(page, pageSize)

	channel := strings.ToLower(strings.TrimSpace(c.Query("channel")))
	status := strings.ToLower(strings.TrimSpace(c.Query("status")))
	eventType := strings.ToLower(strings.TrimSpace(c.Query("event_type")))
	isTestRaw := strings.TrimSpace(c.Query("is_test"))
	createdFromRaw := strings.TrimSpace(c.Query("created_from"))
	createdToRaw := strings.TrimSpace(c.Query("created_to"))

	var isTest *bool
	if isTestRaw != "" {
		parsed, err := strconv.ParseBool(isTestRaw)
		if err != nil {
			shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
			return
		}
		isTest = &parsed
	}

	createdFrom, err := shared.ParseTimeNullable(createdFromRaw)
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}
	createdTo, err := shared.ParseTimeNullable(createdToRaw)
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}

	items, total, err := h.NotificationLogService.ListForAdmin(repository.NotificationLogListFilter{
		Page:        page,
		PageSize:    pageSize,
		Channel:     channel,
		Status:      status,
		EventType:   eventType,
		IsTest:      isTest,
		CreatedFrom: createdFrom,
		CreatedTo:   createdTo,
	})
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.config_fetch_failed", err)
		return
	}

	response.SuccessWithPage(c, items, response.BuildPagination(page, pageSize, total))
}

// TestNotificationCenterSettings ????????
func (h *Handler) TestNotificationCenterSettings(c *gin.Context) {
	var req NotificationCenterTestSendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}
	channel := strings.ToLower(strings.TrimSpace(req.Channel))
	if channel != "email" && channel != "telegram" {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", nil)
		return
	}

	if h.NotificationService == nil {
		shared.RespondError(c, response.CodeInternal, "error.notification_send_failed", nil)
		return
	}

	err := h.NotificationService.SendTest(c.Request.Context(), service.NotificationTestSendInput{
		Channel:   channel,
		Target:    strings.TrimSpace(req.Target),
		Scene:     strings.TrimSpace(req.Scene),
		Locale:    strings.TrimSpace(req.Locale),
		Variables: req.Variables,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNotificationConfigInvalid):
			shared.RespondErrorWithMsg(c, response.CodeBadRequest, err.Error(), nil)
		default:
			shared.RespondError(c, response.CodeInternal, "error.notification_send_failed", err)
		}
		return
	}
	response.Success(c, gin.H{"sent": true})
}
