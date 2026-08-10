package admin

import (
	"errors"
	"strconv"
	"strings"

	"github.com/dujiao-next/internal/http/handlers/shared"
	"github.com/dujiao-next/internal/http/response"
	"github.com/dujiao-next/internal/logger"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/queue"
	"github.com/dujiao-next/internal/service"

	"github.com/gin-gonic/gin"
)

// AdminRefundOrderToWalletRequest ????????????
type AdminRefundOrderToWalletRequest struct {
	Amount string `json:"amount" binding:"required"`
	Remark string `json:"remark"`
}

// AdminManualRefundOrderRequest ?????????(?????/????)
type AdminManualRefundOrderRequest struct {
	Amount string `json:"amount" binding:"required"`
	Remark string `json:"remark"`
}

// GetAdminOrderRefunds ???????????
func (h *Handler) GetAdminOrderRefunds(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	page, pageSize = shared.NormalizePagination(page, pageSize)

	items, total, err := h.OrderRefundService.ListAdminRefundItems(service.AdminOrderRefundListQuery{
		Page:           page,
		PageSize:       pageSize,
		UserID:         c.Query("user_id"),
		UserKeyword:    c.Query("user_keyword"),
		OrderNo:        c.Query("order_no"),
		GuestEmail:     c.Query("guest_email"),
		ProductKeyword: c.Query("product_keyword"),
		ProductName:    c.Query("product_name"),
		CreatedFrom:    c.Query("created_from"),
		CreatedTo:      c.Query("created_to"),
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrOrderFetchFailed):
			shared.RespondError(c, response.CodeInternal, "error.order_fetch_failed", err)
		default:
			shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		}
		return
	}

	pagination := response.BuildPagination(page, pageSize, total)
	response.SuccessWithPage(c, items, pagination)
}

// GetAdminOrderRefund ???????????
func (h *Handler) GetAdminOrderRefund(c *gin.Context) {
	id, err := shared.ParseParamUint(c, "id")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", nil)
		return
	}

	item, err := h.OrderRefundService.GetAdminRefundItem(id)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrOrderNotFound):
			shared.RespondError(c, response.CodeNotFound, "error.order_not_found", nil)
		default:
			shared.RespondError(c, response.CodeInternal, "error.order_fetch_failed", err)
		}
		return
	}
	response.Success(c, item)
}

// AdminRefundOrderToWallet ??????????
func (h *Handler) AdminRefundOrderToWallet(c *gin.Context) {
	orderID, err := shared.ParseParamUint(c, "id")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.order_item_invalid", nil)
		return
	}
	var req AdminRefundOrderToWalletRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}
	amount, err := h.OrderRefundService.ParseRefundAmount(req.Amount)
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}
	order, txn, refundRecord, err := h.WalletService.AdminRefundToWallet(service.AdminRefundToWalletInput{
		OrderID: orderID,
		Amount:  amount,
		Remark:  req.Remark,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrOrderNotFound):
			shared.RespondError(c, response.CodeNotFound, "error.order_not_found", nil)
		case errors.Is(err, service.ErrOrderStatusInvalid):
			shared.RespondError(c, response.CodeBadRequest, "error.order_status_invalid", nil)
		case errors.Is(err, service.ErrOrderRefundExpired):
			shared.RespondError(c, response.CodeBadRequest, "error.order_refund_expired", nil)
		case errors.Is(err, service.ErrWalletInvalidAmount), errors.Is(err, service.ErrWalletRefundExceeded), errors.Is(err, service.ErrWalletNotSupportedForGuest):
			shared.RespondError(c, response.CodeBadRequest, "error.bad_request", nil)
		default:
			shared.RespondError(c, response.CodeInternal, "error.order_update_failed", err)
		}
		return
	}
	h.enqueueOrderRefundStatusEmail(order, refundRecord)

	response.Success(c, gin.H{
		"order":       order,
		"transaction": txn,
	})
}

// AdminManualRefundOrder ???????(?????/????)
func (h *Handler) AdminManualRefundOrder(c *gin.Context) {
	orderID, err := shared.ParseParamUint(c, "id")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.order_item_invalid", nil)
		return
	}
	var req AdminManualRefundOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}
	amount, err := h.OrderRefundService.ParseRefundAmount(req.Amount)
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}
	order, refundRecord, err := h.OrderRefundService.AdminManualRefund(service.AdminManualRefundInput{
		OrderID: orderID,
		Amount:  amount,
		Remark:  req.Remark,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrOrderNotFound):
			shared.RespondError(c, response.CodeNotFound, "error.order_not_found", nil)
		case errors.Is(err, service.ErrOrderStatusInvalid):
			shared.RespondError(c, response.CodeBadRequest, "error.order_status_invalid", nil)
		case errors.Is(err, service.ErrOrderRefundExpired):
			shared.RespondError(c, response.CodeBadRequest, "error.order_refund_expired", nil)
		case errors.Is(err, service.ErrWalletInvalidAmount), errors.Is(err, service.ErrWalletRefundExceeded):
			shared.RespondError(c, response.CodeBadRequest, "error.bad_request", nil)
		default:
			shared.RespondError(c, response.CodeInternal, "error.order_update_failed", err)
		}
		return
	}
	h.enqueueOrderRefundStatusEmail(order, refundRecord)

	response.Success(c, gin.H{
		"order": order,
	})
}

// enqueueOrderRefundStatusEmail ??????????????(???????)?
func (h *Handler) enqueueOrderRefundStatusEmail(order *models.Order, refundRecord *models.OrderRefundRecord) {
	if h == nil || order == nil || h.QueueClient == nil {
		return
	}
	targetOrder := order
	if order.ParentID != nil && *order.ParentID > 0 {
		parentOrder, err := h.OrderRepo.GetByID(*order.ParentID)
		if err != nil {
			logger.Warnw("admin_order_refund_load_parent_failed",
				"order_id", order.ID,
				"parent_id", *order.ParentID,
				"error", err,
			)
		} else if parentOrder != nil {
			targetOrder = parentOrder
		}
	}
	if targetOrder.ID == 0 {
		return
	}
	status := strings.TrimSpace(targetOrder.Status)
	if status == "" {
		return
	}
	if err := h.QueueClient.EnqueueOrderStatusEmail(queue.OrderStatusEmailPayload{
		OrderID:        targetOrder.ID,
		Status:         status,
		RefundRecordID: resolveRefundRecordID(refundRecord),
	}); err != nil {
		logger.Warnw("admin_order_refund_enqueue_status_email_failed",
			"order_id", targetOrder.ID,
			"status", status,
			"error", err,
		)
	}
}

// resolveRefundRecordID ????????ID?
func resolveRefundRecordID(record *models.OrderRefundRecord) uint {
	if record == nil {
		return 0
	}
	return record.ID
}
