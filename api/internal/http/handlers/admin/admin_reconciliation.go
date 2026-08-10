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

// RunReconciliation ??????
func (h *Handler) RunReconciliation(c *gin.Context) {
	if h.ReconciliationService == nil {
		shared.RespondErrorWithMsg(c, response.CodeInternal, "service not available", nil)
		return
	}
	var input service.RunReconciliationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}

	job, err := h.ReconciliationService.CreateAndEnqueue(input)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.reconciliation_create_failed", err)
		return
	}

	response.Success(c, job)
}

// GetReconciliationJobs ??????
func (h *Handler) GetReconciliationJobs(c *gin.Context) {
	if h.ReconciliationService == nil {
		shared.RespondErrorWithMsg(c, response.CodeInternal, "service not available", nil)
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	page, pageSize = shared.NormalizePagination(page, pageSize)

	filter := repository.ReconciliationJobListFilter{
		Pagination: repository.Pagination{Page: page, PageSize: pageSize},
	}
	if connID := strings.TrimSpace(c.Query("connection_id")); connID != "" {
		if id, err := shared.ParseQueryUint(connID, false); err == nil {
			filter.ConnectionID = id
		}
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		filter.Status = status
	}
	if t := strings.TrimSpace(c.Query("type")); t != "" {
		filter.Type = t
	}

	jobs, total, err := h.ReconciliationService.ListJobs(filter)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.reconciliation_fetch_failed", err)
		return
	}
	pagination := response.BuildPagination(page, pageSize, total)
	response.SuccessWithPage(c, jobs, pagination)
}

// GetReconciliationJob ??????
func (h *Handler) GetReconciliationJob(c *gin.Context) {
	if h.ReconciliationService == nil {
		shared.RespondErrorWithMsg(c, response.CodeInternal, "service not available", nil)
		return
	}
	id, err := shared.ParseParamUint(c, "id")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}

	job, err := h.ReconciliationService.GetJob(id)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.reconciliation_fetch_failed", err)
		return
	}

	// ?????
	itemPage, _ := strconv.Atoi(c.DefaultQuery("items_page", "1"))
	itemPageSize, _ := strconv.Atoi(c.DefaultQuery("items_page_size", "20"))
	itemPage, itemPageSize = shared.NormalizePagination(itemPage, itemPageSize)

	items, itemsTotal, err := h.ReconciliationService.GetJobItems(id, itemPage, itemPageSize)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.reconciliation_fetch_failed", err)
		return
	}

	response.Success(c, gin.H{
		"job":         job,
		"items":       items,
		"items_total": itemsTotal,
	})
}

// ResolveReconciliationItem ???????
func (h *Handler) ResolveReconciliationItem(c *gin.Context) {
	if h.ReconciliationService == nil {
		shared.RespondErrorWithMsg(c, response.CodeInternal, "service not available", nil)
		return
	}
	id, err := shared.ParseParamUint(c, "id")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}

	var input struct {
		Remark string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}

	adminID, ok := shared.GetAdminID(c)
	if !ok {
		shared.RespondError(c, response.CodeUnauthorized, "error.unauthorized", nil)
		return
	}
	if err := h.ReconciliationService.ResolveItem(id, adminID, input.Remark); err != nil {
		if errors.Is(err, service.ErrReconciliationItemNotFound) {
			shared.RespondError(c, response.CodeNotFound, "error.reconciliation_item_not_found", nil)
			return
		}
		shared.RespondError(c, response.CodeInternal, "error.reconciliation_resolve_failed", err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}
