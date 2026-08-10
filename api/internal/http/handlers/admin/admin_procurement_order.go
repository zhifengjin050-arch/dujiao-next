package admin

import (
	"errors"
	"strconv"
	"strings"

	"github.com/dujiao-next/internal/http/handlers/shared"
	"github.com/dujiao-next/internal/http/response"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/repository"
	"github.com/dujiao-next/internal/service"

	"github.com/gin-gonic/gin"
)

// GetProcurementOrders ?????
func (h *Handler) GetProcurementOrders(c *gin.Context) {
	if h.ProcurementOrderService == nil {
		shared.RespondErrorWithMsg(c, response.CodeInternal, "service not available", nil)
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	page, pageSize = shared.NormalizePagination(page, pageSize)

	filter := repository.ProcurementOrderListFilter{
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
	if orderNo := strings.TrimSpace(c.Query("order_no")); orderNo != "" {
		filter.LocalOrderNo = orderNo
	}
	if upstreamOrderNo := strings.TrimSpace(c.Query("upstream_order_no")); upstreamOrderNo != "" {
		filter.UpstreamOrderNo = upstreamOrderNo
	}
	if createdFrom := strings.TrimSpace(c.Query("created_from")); createdFrom != "" {
		filter.CreatedFrom = createdFrom
	}
	if createdTo := strings.TrimSpace(c.Query("created_to")); createdTo != "" {
		filter.CreatedTo = createdTo
	}

	orders, total, err := h.ProcurementOrderService.List(filter)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.procurement_fetch_failed", err)
		return
	}

	pagination := response.BuildPagination(page, pageSize, total)
	response.SuccessWithPage(c, orders, pagination)
}

// GetProcurementOrder ?????
func (h *Handler) GetProcurementOrder(c *gin.Context) {
	if h.ProcurementOrderService == nil {
		shared.RespondErrorWithMsg(c, response.CodeInternal, "service not available", nil)
		return
	}
	id, err := shared.ParseParamUint(c, "id")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}
	order, err := h.ProcurementOrderService.GetByID(id)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.procurement_fetch_failed", err)
		return
	}
	if order == nil {
		shared.RespondError(c, response.CodeNotFound, "error.procurement_not_found", nil)
		return
	}
	order.TruncateUpstreamPayload(models.FulfillmentPayloadMaxPreviewLines)
	h.ProcurementOrderService.FillParentOrderNo(order)
	response.Success(c, order)
}

// DownloadProcurementUpstreamPayload ???????????
func (h *Handler) DownloadProcurementUpstreamPayload(c *gin.Context) {
	if h.ProcurementOrderService == nil {
		shared.RespondErrorWithMsg(c, response.CodeInternal, "service not available", nil)
		return
	}
	id, err := shared.ParseParamUint(c, "id")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}
	order, err := h.ProcurementOrderService.GetByID(id)
	if err != nil || order == nil {
		shared.RespondError(c, response.CodeNotFound, "error.procurement_not_found", nil)
		return
	}
	if order.UpstreamPayload == "" {
		shared.RespondError(c, response.CodeNotFound, "error.fulfillment_not_found", nil)
		return
	}
	filename := "upstream-payload-" + strconv.FormatUint(uint64(order.ID), 10) + ".txt"
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Data(200, "text/plain; charset=utf-8", []byte(order.UpstreamPayload))
}

// RetryProcurementOrder ???????
func (h *Handler) RetryProcurementOrder(c *gin.Context) {
	if h.ProcurementOrderService == nil {
		shared.RespondErrorWithMsg(c, response.CodeInternal, "service not available", nil)
		return
	}
	id, err := shared.ParseParamUint(c, "id")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}
	if err := h.ProcurementOrderService.RetryManual(id); err != nil {
		if errors.Is(err, service.ErrProcurementNotFound) {
			shared.RespondError(c, response.CodeNotFound, "error.procurement_not_found", nil)
			return
		}
		if errors.Is(err, service.ErrProcurementStatusInvalid) {
			shared.RespondErrorWithMsg(c, response.CodeBadRequest, err.Error(), nil)
			return
		}
		shared.RespondError(c, response.CodeInternal, "error.procurement_retry_failed", err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// CancelProcurementOrder ???????
func (h *Handler) CancelProcurementOrder(c *gin.Context) {
	if h.ProcurementOrderService == nil {
		shared.RespondErrorWithMsg(c, response.CodeInternal, "service not available", nil)
		return
	}
	id, err := shared.ParseParamUint(c, "id")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}
	if err := h.ProcurementOrderService.CancelManual(id); err != nil {
		if errors.Is(err, service.ErrProcurementNotFound) {
			shared.RespondError(c, response.CodeNotFound, "error.procurement_not_found", nil)
			return
		}
		if errors.Is(err, service.ErrProcurementStatusInvalid) {
			shared.RespondErrorWithMsg(c, response.CodeBadRequest, err.Error(), nil)
			return
		}
		shared.RespondError(c, response.CodeInternal, "error.procurement_cancel_failed", err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}
