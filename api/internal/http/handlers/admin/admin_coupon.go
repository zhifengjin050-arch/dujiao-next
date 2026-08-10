package admin

import (
	"errors"
	"strconv"

	"github.com/dujiao-next/internal/http/handlers/shared"
	"github.com/dujiao-next/internal/http/response"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/repository"
	"github.com/dujiao-next/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

// CreateCouponRequest ???????
type CreateCouponRequest struct {
	Code         string   `json:"code" binding:"required"`
	Type         string   `json:"type" binding:"required"`
	Value        float64  `json:"value" binding:"required"`
	MinAmount    float64  `json:"min_amount"`
	MaxDiscount  float64  `json:"max_discount"`
	UsageLimit   int      `json:"usage_limit"`
	PerUserLimit int      `json:"per_user_limit"`
	PaymentRoles []string `json:"payment_roles"`
	MemberLevels []uint   `json:"member_levels"`
	ScopeRefIDs  []uint   `json:"scope_ref_ids" binding:"required"`
	StartsAt     string   `json:"starts_at"`
	EndsAt       string   `json:"ends_at"`
	IsActive     *bool    `json:"is_active"`
}

// CreateCoupon ?????
func (h *Handler) CreateCoupon(c *gin.Context) {
	var req CreateCouponRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}

	startsAt, err := shared.ParseTimeNullable(req.StartsAt)
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}
	endsAt, err := shared.ParseTimeNullable(req.EndsAt)
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}

	coupon, err := h.CouponAdminService.Create(service.CreateCouponInput{
		Code:         req.Code,
		Type:         req.Type,
		Value:        models.NewMoneyFromDecimal(decimal.NewFromFloat(req.Value)),
		MinAmount:    models.NewMoneyFromDecimal(decimal.NewFromFloat(req.MinAmount)),
		MaxDiscount:  models.NewMoneyFromDecimal(decimal.NewFromFloat(req.MaxDiscount)),
		UsageLimit:   req.UsageLimit,
		PerUserLimit: req.PerUserLimit,
		PaymentRoles: req.PaymentRoles,
		MemberLevels: req.MemberLevels,
		ScopeRefIDs:  req.ScopeRefIDs,
		StartsAt:     startsAt,
		EndsAt:       endsAt,
		IsActive:     req.IsActive,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrCouponInvalid):
			shared.RespondError(c, response.CodeBadRequest, "error.coupon_invalid", nil)
		case errors.Is(err, service.ErrCouponScopeInvalid):
			shared.RespondError(c, response.CodeBadRequest, "error.coupon_scope_invalid", nil)
		default:
			shared.RespondError(c, response.CodeInternal, "error.coupon_create_failed", err)
		}
		return
	}

	response.Success(c, coupon)
}

// UpdateCoupon ?????
func (h *Handler) UpdateCoupon(c *gin.Context) {
	couponID, err := shared.ParseParamUint(c, "id")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}
	var req CreateCouponRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}

	startsAt, err := shared.ParseTimeNullable(req.StartsAt)
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}
	endsAt, err := shared.ParseTimeNullable(req.EndsAt)
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}

	coupon, err := h.CouponAdminService.Update(couponID, service.UpdateCouponInput{
		Code:         req.Code,
		Type:         req.Type,
		Value:        models.NewMoneyFromDecimal(decimal.NewFromFloat(req.Value)),
		MinAmount:    models.NewMoneyFromDecimal(decimal.NewFromFloat(req.MinAmount)),
		MaxDiscount:  models.NewMoneyFromDecimal(decimal.NewFromFloat(req.MaxDiscount)),
		UsageLimit:   req.UsageLimit,
		PerUserLimit: req.PerUserLimit,
		PaymentRoles: req.PaymentRoles,
		MemberLevels: req.MemberLevels,
		ScopeRefIDs:  req.ScopeRefIDs,
		StartsAt:     startsAt,
		EndsAt:       endsAt,
		IsActive:     req.IsActive,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrCouponNotFound):
			shared.RespondError(c, response.CodeNotFound, "error.coupon_not_found", nil)
		case errors.Is(err, service.ErrCouponInvalid):
			shared.RespondError(c, response.CodeBadRequest, "error.coupon_invalid", nil)
		case errors.Is(err, service.ErrCouponScopeInvalid):
			shared.RespondError(c, response.CodeBadRequest, "error.coupon_scope_invalid", nil)
		default:
			shared.RespondError(c, response.CodeInternal, "error.coupon_update_failed", err)
		}
		return
	}

	response.Success(c, coupon)
}

// DeleteCoupon ?????
func (h *Handler) DeleteCoupon(c *gin.Context) {
	couponID, err := shared.ParseParamUint(c, "id")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}
	if err := h.CouponAdminService.Delete(couponID); err != nil {
		switch {
		case errors.Is(err, service.ErrCouponNotFound):
			shared.RespondError(c, response.CodeNotFound, "error.coupon_not_found", nil)
		case errors.Is(err, service.ErrCouponInvalid):
			shared.RespondError(c, response.CodeBadRequest, "error.coupon_invalid", nil)
		default:
			shared.RespondError(c, response.CodeInternal, "error.coupon_delete_failed", err)
		}
		return
	}
	response.Success(c, gin.H{
		"deleted": true,
	})
}

// GetAdminCoupons ???????
func (h *Handler) GetAdminCoupons(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	page, pageSize = shared.NormalizePagination(page, pageSize)

	code := c.Query("code")
	id, err := shared.ParseQueryUint(c.Query("id"), true)
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}
	scopeRefID, err := shared.ParseQueryUint(c.Query("scope_ref_id"), true)
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}
	var isActive *bool
	if raw := c.Query("is_active"); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
			return
		}
		isActive = &parsed
	}

	coupons, total, err := h.CouponAdminService.List(repository.CouponListFilter{
		ID:         id,
		Code:       code,
		ScopeRefID: scopeRefID,
		IsActive:   isActive,
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.coupon_fetch_failed", err)
		return
	}

	pagination := response.BuildPagination(page, pageSize, total)
	response.SuccessWithPage(c, coupons, pagination)
}
