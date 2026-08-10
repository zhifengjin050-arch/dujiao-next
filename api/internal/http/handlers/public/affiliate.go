package public

import (
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"github.com/dujiao-next/internal/dto"
	"github.com/dujiao-next/internal/http/handlers/shared"
	"github.com/dujiao-next/internal/http/response"
	"github.com/dujiao-next/internal/service"
)

// AffiliateTrackClickRequest ????????
type AffiliateTrackClickRequest struct {
	AffiliateCode string `json:"affiliate_code" binding:"required"`
	VisitorKey    string `json:"visitor_key"`
	LandingPath   string `json:"landing_path"`
	Referrer      string `json:"referrer"`
}

// TrackAffiliateClick ??????
func (h *Handler) TrackAffiliateClick(c *gin.Context) {
	var req AffiliateTrackClickRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}

	if h.AffiliateService != nil {
		if err := h.AffiliateService.TrackClick(service.AffiliateTrackClickInput{
			AffiliateCode: req.AffiliateCode,
			VisitorKey:    req.VisitorKey,
			LandingPath:   req.LandingPath,
			Referrer:      req.Referrer,
			ClientIP:      c.ClientIP(),
			UserAgent:     c.GetHeader("User-Agent"),
		}); err != nil {
			shared.RespondError(c, response.CodeInternal, "error.save_failed", err)
			return
		}
	}
	response.Success(c, gin.H{"ok": true})
}

// OpenAffiliate ??????
func (h *Handler) OpenAffiliate(c *gin.Context) {
	uid, ok := shared.GetUserID(c)
	if !ok {
		return
	}
	if h.AffiliateService == nil {
		shared.RespondError(c, response.CodeInternal, "error.save_failed", nil)
		return
	}

	profile, err := h.AffiliateService.OpenAffiliate(uid)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAffiliateDisabled):
			shared.RespondError(c, response.CodeBadRequest, "error.forbidden", nil)
		case errors.Is(err, service.ErrNotFound):
			shared.RespondError(c, response.CodeNotFound, "error.user_not_found", nil)
		default:
			shared.RespondError(c, response.CodeInternal, "error.save_failed", err)
		}
		return
	}
	response.Success(c, dto.NewAffiliateProfileResp(profile))
}

// GetAffiliateDashboard ????????
func (h *Handler) GetAffiliateDashboard(c *gin.Context) {
	uid, ok := shared.GetUserID(c)
	if !ok {
		return
	}
	if h.AffiliateService == nil {
		shared.RespondError(c, response.CodeInternal, "error.user_fetch_failed", nil)
		return
	}
	data, err := h.AffiliateService.GetUserDashboard(uid)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.user_fetch_failed", err)
		return
	}
	response.Success(c, data)
}

// ListAffiliateCommissions ??????????
func (h *Handler) ListAffiliateCommissions(c *gin.Context) {
	uid, ok := shared.GetUserID(c)
	if !ok {
		return
	}
	if h.AffiliateService == nil {
		shared.RespondError(c, response.CodeInternal, "error.user_fetch_failed", nil)
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	page, pageSize = shared.NormalizePagination(page, pageSize)
	status := strings.TrimSpace(c.Query("status"))

	rows, total, err := h.AffiliateService.ListUserCommissions(uid, page, pageSize, status)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.user_fetch_failed", err)
		return
	}
	response.SuccessWithPage(c, dto.NewAffiliateCommissionRespList(rows), response.BuildPagination(page, pageSize, total))
}

// ListAffiliateWithdraws ??????????
func (h *Handler) ListAffiliateWithdraws(c *gin.Context) {
	uid, ok := shared.GetUserID(c)
	if !ok {
		return
	}
	if h.AffiliateService == nil {
		shared.RespondError(c, response.CodeInternal, "error.user_fetch_failed", nil)
		return
	}
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	page, pageSize = shared.NormalizePagination(page, pageSize)
	status := strings.TrimSpace(c.Query("status"))

	rows, total, err := h.AffiliateService.ListUserWithdraws(uid, page, pageSize, status)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.user_fetch_failed", err)
		return
	}
	response.SuccessWithPage(c, dto.NewAffiliateWithdrawRespList(rows), response.BuildPagination(page, pageSize, total))
}

// AffiliateWithdrawApplyRequest ??????
type AffiliateWithdrawApplyRequest struct {
	Amount  string `json:"amount" binding:"required"`
	Channel string `json:"channel" binding:"required"`
	Account string `json:"account" binding:"required"`
}

// ApplyAffiliateWithdraw ??????
func (h *Handler) ApplyAffiliateWithdraw(c *gin.Context) {
	uid, ok := shared.GetUserID(c)
	if !ok {
		return
	}
	if h.AffiliateService == nil {
		shared.RespondError(c, response.CodeInternal, "error.save_failed", nil)
		return
	}

	var req AffiliateWithdrawApplyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}
	amount, err := decimal.NewFromString(strings.TrimSpace(req.Amount))
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", nil)
		return
	}

	row, err := h.AffiliateService.ApplyWithdraw(uid, service.AffiliateWithdrawApplyInput{
		Amount:  amount,
		Channel: req.Channel,
		Account: req.Account,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrAffiliateDisabled):
			shared.RespondError(c, response.CodeBadRequest, "error.forbidden", nil)
		case errors.Is(err, service.ErrAffiliateNotOpened):
			shared.RespondError(c, response.CodeBadRequest, "error.bad_request", nil)
		case errors.Is(err, service.ErrAffiliateWithdrawAmountInvalid):
			shared.RespondError(c, response.CodeBadRequest, "error.bad_request", nil)
		case errors.Is(err, service.ErrAffiliateWithdrawChannelInvalid):
			shared.RespondError(c, response.CodeBadRequest, "error.bad_request", nil)
		case errors.Is(err, service.ErrAffiliateWithdrawInsufficient):
			shared.RespondError(c, response.CodeBadRequest, "error.bad_request", nil)
		default:
			shared.RespondError(c, response.CodeInternal, "error.save_failed", err)
		}
		return
	}
	response.Success(c, dto.NewAffiliateWithdrawResp(row))
}
