package admin

import (
	"strconv"
	"strings"

	"github.com/dujiao-next/internal/http/handlers/shared"
	"github.com/dujiao-next/internal/http/response"
	"github.com/dujiao-next/internal/repository"

	"github.com/gin-gonic/gin"
)

// GetUserLoginLogs ??????????
func (h *Handler) GetUserLoginLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	page, pageSize = shared.NormalizePagination(page, pageSize)

	userIDRaw := c.Query("user_id")
	email := strings.TrimSpace(c.Query("email"))
	status := strings.TrimSpace(c.Query("status"))
	failReason := strings.TrimSpace(c.Query("fail_reason"))
	clientIP := strings.TrimSpace(c.Query("client_ip"))
	createdFromRaw := strings.TrimSpace(c.Query("created_from"))
	createdToRaw := strings.TrimSpace(c.Query("created_to"))

	var userID uint
	if userIDRaw != "" {
		parsedUserID, err := shared.ParseQueryUint(userIDRaw, false)
		if err != nil {
			shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
			return
		}
		userID = parsedUserID
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

	logs, total, err := h.UserLoginLogService.ListForAdmin(repository.UserLoginLogListFilter{
		Page:        page,
		PageSize:    pageSize,
		UserID:      userID,
		Email:       email,
		Status:      status,
		FailReason:  failReason,
		ClientIP:    clientIP,
		CreatedFrom: createdFrom,
		CreatedTo:   createdTo,
	})
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.user_login_log_fetch_failed", err)
		return
	}

	pagination := response.BuildPagination(page, pageSize, total)
	response.SuccessWithPage(c, logs, pagination)
}
