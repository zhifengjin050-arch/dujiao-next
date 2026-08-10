package admin

import (
	"strconv"
	"strings"

	"github.com/dujiao-next/internal/http/handlers/shared"
	"github.com/dujiao-next/internal/http/response"
	"github.com/dujiao-next/internal/repository"

	"github.com/gin-gonic/gin"
)

// ListAuthzAuditLogs ??????????
func (h *Handler) ListAuthzAuditLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	page, pageSize = shared.NormalizePagination(page, pageSize)

	operatorAdminIDRaw := c.Query("operator_admin_id")
	targetAdminIDRaw := c.Query("target_admin_id")
	action := strings.TrimSpace(c.Query("action"))
	role := strings.TrimSpace(c.Query("role"))
	object := strings.TrimSpace(c.Query("object"))
	method := strings.TrimSpace(c.Query("method"))
	createdFromRaw := strings.TrimSpace(c.Query("created_from"))
	createdToRaw := strings.TrimSpace(c.Query("created_to"))

	var operatorAdminID uint
	if operatorAdminIDRaw != "" {
		parsedOperatorAdminID, err := shared.ParseQueryUint(operatorAdminIDRaw, false)
		if err != nil {
			shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
			return
		}
		operatorAdminID = parsedOperatorAdminID
	}

	var targetAdminID uint
	if targetAdminIDRaw != "" {
		parsedTargetAdminID, err := shared.ParseQueryUint(targetAdminIDRaw, false)
		if err != nil {
			shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
			return
		}
		targetAdminID = parsedTargetAdminID
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

	items, total, err := h.AuthzAuditService.ListForAdmin(repository.AuthzAuditLogListFilter{
		Page:            page,
		PageSize:        pageSize,
		OperatorAdminID: operatorAdminID,
		TargetAdminID:   targetAdminID,
		Action:          action,
		Role:            role,
		Object:          object,
		Method:          method,
		CreatedFrom:     createdFrom,
		CreatedTo:       createdTo,
	})
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.config_fetch_failed", err)
		return
	}

	pagination := response.BuildPagination(page, pageSize, total)
	response.SuccessWithPage(c, items, pagination)
}
