package admin

import (
	"errors"
	"strconv"
	"strings"

	"github.com/dujiao-next/internal/http/handlers/shared"
	"github.com/dujiao-next/internal/http/response"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/service"

	"github.com/gin-gonic/gin"
)

type createTelegramBroadcastRequest struct {
	Title          string      `json:"title" binding:"required"`
	RecipientType  string      `json:"recipient_type" binding:"required"`
	UserIDs        []uint      `json:"user_ids"`
	Filters        models.JSON `json:"filters"`
	AttachmentURL  string      `json:"attachment_url"`
	AttachmentName string      `json:"attachment_name"`
	MessageHTML    string      `json:"message_html" binding:"required"`
}

// ListTelegramBroadcasts ?? Telegram ?????
func (h *Handler) ListTelegramBroadcasts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	page, pageSize = shared.NormalizePagination(page, pageSize)

	items, total, err := h.TelegramBroadcastService.ListBroadcasts(service.TelegramBroadcastListInput{
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.bad_request", err)
		return
	}
	response.SuccessWithPage(c, items, response.BuildPagination(page, pageSize, total))
}

// CreateTelegramBroadcast ?? Telegram ?????
func (h *Handler) CreateTelegramBroadcast(c *gin.Context) {
	var req createTelegramBroadcastRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}

	result, err := h.TelegramBroadcastService.CreateBroadcast(c.Request.Context(), service.TelegramBroadcastCreateInput{
		Title:          req.Title,
		RecipientType:  req.RecipientType,
		UserIDs:        req.UserIDs,
		Filters:        req.Filters,
		AttachmentURL:  req.AttachmentURL,
		AttachmentName: req.AttachmentName,
		MessageHTML:    req.MessageHTML,
	})
	if err != nil {
		if errors.Is(err, service.ErrTelegramBroadcastInvalid) ||
			errors.Is(err, service.ErrTelegramBroadcastNoRecipients) ||
			errors.Is(err, service.ErrTelegramBotTokenUnavailable) {
			shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
			return
		}
		shared.RespondError(c, response.CodeInternal, "error.bad_request", err)
		return
	}
	response.Success(c, result)
}

// GetTelegramBroadcast ???? Telegram ?????
func (h *Handler) GetTelegramBroadcast(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", errors.New("invalid broadcast id"))
		return
	}
	broadcast, err := h.TelegramBroadcastService.GetBroadcast(uint(id))
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.bad_request", err)
		return
	}
	if broadcast == nil {
		shared.RespondError(c, response.CodeNotFound, "error.not_found", service.ErrTelegramBroadcastNotFound)
		return
	}
	response.Success(c, broadcast)
}

// ListTelegramBroadcastUsers ?? Telegram ???????
func (h *Handler) ListTelegramBroadcastUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	page, pageSize = shared.NormalizePagination(page, pageSize)

	createdFrom, err := shared.ParseTimeNullable(strings.TrimSpace(c.Query("created_from")))
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}
	createdTo, err := shared.ParseTimeNullable(strings.TrimSpace(c.Query("created_to")))
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}

	items, total, err := h.TelegramBroadcastService.ListTelegramUsers(service.TelegramBroadcastUserQuery{
		Page:             page,
		PageSize:         pageSize,
		Keyword:          strings.TrimSpace(c.Query("keyword")),
		DisplayName:      strings.TrimSpace(c.Query("display_name")),
		TelegramUsername: strings.TrimSpace(c.Query("telegram_username")),
		TelegramUserID:   strings.TrimSpace(c.Query("telegram_user_id")),
		CreatedFrom:      createdFrom,
		CreatedTo:        createdTo,
	})
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.bad_request", err)
		return
	}
	response.SuccessWithPage(c, items, response.BuildPagination(page, pageSize, total))
}
