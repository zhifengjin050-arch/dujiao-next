package admin

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/http/handlers/shared"
	"github.com/dujiao-next/internal/http/response"
	"github.com/dujiao-next/internal/repository"
	"github.com/dujiao-next/internal/service"
)

// AffiliateProfileStatusRequest ??????????
type AffiliateProfileStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

// BatchAffiliateProfileStatusRequest ????????????
type BatchAffiliateProfileStatusRequest struct {
	ProfileIDs []uint `json:"profile_ids" binding:"required"`
	Status     string `json:"status" binding:"required"`
}

// ListAffiliateUsers ?????????
func (h *Handler) ListAffiliateUsers(c *gin.Context) {
	if h.AffiliateService == nil {
		shared.RespondError(c, response.CodeInternal, "error.user_fetch_failed", nil)
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	page, pageSize = shared.NormalizePagination(page, pageSize)
	userID, _ := shared.ParseQueryUint(c.Query("user_id"), false)

	rows, total, err := h.AffiliateService.ListAdminUsers(repository.AffiliateProfileListFilter{
		Page:     page,
		PageSize: pageSize,
		UserID:   userID,
		Status:   strings.TrimSpace(c.Query("status")),
		Code:     strings.TrimSpace(c.Query("code")),
		Keyword:  strings.TrimSpace(c.Query("keyword")),
	})
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.user_fetch_failed", err)
		return
	}
	response.SuccessWithPage(c, rows, response.BuildPagination(page, pageSize, total))
}

// ListAffiliateCommissions ???????
func (h *Handler) ListAffiliateCommissions(c *gin.Context) {
	if h.AffiliateService == nil {
		shared.RespondError(c, response.CodeInternal, "error.user_fetch_failed", nil)
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	page, pageSize = shared.NormalizePagination(page, pageSize)
	profileID, _ := shared.ParseQueryUint(c.Query("affiliate_profile_id"), false)

	rows, total, err := h.AffiliateService.ListAdminCommissions(service.AffiliateAdminCommissionListFilter{
		Page:               page,
		PageSize:           pageSize,
		AffiliateProfileID: profileID,
		OrderNo:            strings.TrimSpace(c.Query("order_no")),
		Status:             strings.TrimSpace(c.Query("status")),
		Keyword:            strings.TrimSpace(c.Query("keyword")),
	})
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.user_fetch_failed", err)
		return
	}
	response.SuccessWithPage(c, rows, response.BuildPagination(page, pageSize, total))
}

// ListAffiliateWithdraws ?????????
func (h *Handler) ListAffiliateWithdraws(c *gin.Context) {
	if h.AffiliateService == nil {
		shared.RespondError(c, response.CodeInternal, "error.user_fetch_failed", nil)
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	page, pageSize = shared.NormalizePagination(page, pageSize)
	profileID, _ := shared.ParseQueryUint(c.Query("affiliate_profile_id"), false)

	rows, total, err := h.AffiliateService.ListAdminWithdraws(service.AffiliateAdminWithdrawListFilter{
		Page:               page,
		PageSize:           pageSize,
		AffiliateProfileID: profileID,
		Status:             strings.TrimSpace(c.Query("status")),
		Keyword:            strings.TrimSpace(c.Query("keyword")),
	})
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.user_fetch_failed", err)
		return
	}
	response.SuccessWithPage(c, rows, response.BuildPagination(page, pageSize, total))
}

// UpdateAffiliateUserStatus ???????????
func (h *Handler) UpdateAffiliateUserStatus(c *gin.Context) {
	if h.AffiliateService == nil {
		shared.RespondError(c, response.CodeInternal, "error.save_failed", nil)
		return
	}
	id, err := shared.ParseParamUint(c, "id")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", nil)
		return
	}

	var req AffiliateProfileStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}

	row, err := h.AffiliateService.UpdateAffiliateProfileStatus(id, strings.TrimSpace(req.Status))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNotFound):
			shared.RespondError(c, response.CodeNotFound, "error.bad_request", nil)
		case errors.Is(err, service.ErrAffiliateProfileStatusInvalid):
			shared.RespondError(c, response.CodeBadRequest, "error.bad_request", nil)
		default:
			shared.RespondError(c, response.CodeInternal, "error.save_failed", err)
		}
		return
	}
	response.Success(c, row)
}

// BatchUpdateAffiliateUserStatus ?????????????
func (h *Handler) BatchUpdateAffiliateUserStatus(c *gin.Context) {
	if h.AffiliateService == nil {
		shared.RespondError(c, response.CodeInternal, "error.save_failed", nil)
		return
	}
	var req BatchAffiliateProfileStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}
	if len(req.ProfileIDs) == 0 {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", nil)
		return
	}
	updated, err := h.AffiliateService.BatchUpdateAffiliateProfileStatus(req.ProfileIDs, strings.TrimSpace(req.Status))
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAffiliateProfileStatusInvalid):
			shared.RespondError(c, response.CodeBadRequest, "error.bad_request", nil)
		default:
			shared.RespondError(c, response.CodeInternal, "error.save_failed", err)
		}
		return
	}
	response.Success(c, gin.H{"updated": updated})
}

// AffiliateReviewWithdrawRequest ??????
type AffiliateReviewWithdrawRequest struct {
	Reason string `json:"reason"`
}

// RejectAffiliateWithdraw ??????
func (h *Handler) RejectAffiliateWithdraw(c *gin.Context) {
	adminID, ok := shared.GetAdminID(c)
	if !ok {
		return
	}
	if h.AffiliateService == nil {
		shared.RespondError(c, response.CodeInternal, "error.save_failed", nil)
		return
	}
	id, err := shared.ParseParamUint(c, "id")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", nil)
		return
	}

	var req AffiliateReviewWithdrawRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}
	row, err := h.AffiliateService.ReviewWithdraw(adminID, id, constants.AffiliateWithdrawActionReject, req.Reason)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNotFound):
			shared.RespondError(c, response.CodeNotFound, "error.bad_request", nil)
		case errors.Is(err, service.ErrAffiliateWithdrawStatusInvalid):
			shared.RespondError(c, response.CodeBadRequest, "error.bad_request", nil)
		default:
			shared.RespondError(c, response.CodeInternal, "error.save_failed", err)
		}
		return
	}
	response.Success(c, row)
}

// PayAffiliateWithdraw ???????
func (h *Handler) PayAffiliateWithdraw(c *gin.Context) {
	adminID, ok := shared.GetAdminID(c)
	if !ok {
		return
	}
	if h.AffiliateService == nil {
		shared.RespondError(c, response.CodeInternal, "error.save_failed", nil)
		return
	}
	id, err := shared.ParseParamUint(c, "id")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", nil)
		return
	}
	row, err := h.AffiliateService.ReviewWithdraw(adminID, id, constants.AffiliateWithdrawActionPay, "")
	if err != nil {
		switch {
		case errors.Is(err, service.ErrNotFound):
			shared.RespondError(c, response.CodeNotFound, "error.bad_request", nil)
		case errors.Is(err, service.ErrAffiliateWithdrawStatusInvalid):
			shared.RespondError(c, response.CodeBadRequest, "error.bad_request", nil)
		default:
			shared.RespondError(c, response.CodeInternal, "error.save_failed", err)
		}
		return
	}
	response.Success(c, row)
}
