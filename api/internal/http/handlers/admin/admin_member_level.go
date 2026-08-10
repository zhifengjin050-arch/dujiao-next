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

// CreateMemberLevelRequest ??/????????
type CreateMemberLevelRequest struct {
	NameJSON          models.JSON `json:"name" binding:"required"`
	Slug              string      `json:"slug" binding:"required"`
	Icon              string      `json:"icon"`
	DiscountRate      float64     `json:"discount_rate"`
	RechargeThreshold float64     `json:"recharge_threshold"`
	SpendThreshold    float64     `json:"spend_threshold"`
	IsDefault         bool        `json:"is_default"`
	SortOrder         int         `json:"sort_order"`
	IsActive          *bool       `json:"is_active"`
}

// GetAdminMemberLevels ????????
func (h *Handler) GetAdminMemberLevels(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "50"))
	page, pageSize = shared.NormalizePagination(page, pageSize)

	var isActive *bool
	if raw := c.Query("is_active"); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
			return
		}
		isActive = &parsed
	}

	levels, total, err := h.MemberLevelService.ListLevels(repository.MemberLevelListFilter{
		IsActive: isActive,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.member_level_fetch_failed", err)
		return
	}

	pagination := response.BuildPagination(page, pageSize, total)
	response.SuccessWithPage(c, levels, pagination)
}

// CreateMemberLevel ??????
func (h *Handler) CreateMemberLevel(c *gin.Context) {
	var req CreateMemberLevelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	level := &models.MemberLevel{
		NameJSON:          req.NameJSON,
		Slug:              req.Slug,
		Icon:              req.Icon,
		DiscountRate:      models.NewMoneyFromDecimal(decimal.NewFromFloat(req.DiscountRate)),
		RechargeThreshold: models.NewMoneyFromDecimal(decimal.NewFromFloat(req.RechargeThreshold)),
		SpendThreshold:    models.NewMoneyFromDecimal(decimal.NewFromFloat(req.SpendThreshold)),
		IsDefault:         req.IsDefault,
		SortOrder:         req.SortOrder,
		IsActive:          isActive,
	}

	if err := h.MemberLevelService.CreateLevel(level); err != nil {
		switch {
		case errors.Is(err, service.ErrMemberLevelSlugExists):
			shared.RespondError(c, response.CodeBadRequest, "error.member_level_slug_exists", nil)
		default:
			shared.RespondError(c, response.CodeInternal, "error.member_level_create_failed", err)
		}
		return
	}

	response.Success(c, level)
}

// UpdateMemberLevel ??????
func (h *Handler) UpdateMemberLevel(c *gin.Context) {
	levelID, err := shared.ParseParamUint(c, "id")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}

	existing, err := h.MemberLevelService.GetByID(levelID)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.member_level_fetch_failed", err)
		return
	}
	if existing == nil {
		shared.RespondError(c, response.CodeNotFound, "error.member_level_not_found", nil)
		return
	}

	var req CreateMemberLevelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}

	existing.NameJSON = req.NameJSON
	existing.Slug = req.Slug
	existing.Icon = req.Icon
	existing.DiscountRate = models.NewMoneyFromDecimal(decimal.NewFromFloat(req.DiscountRate))
	existing.RechargeThreshold = models.NewMoneyFromDecimal(decimal.NewFromFloat(req.RechargeThreshold))
	existing.SpendThreshold = models.NewMoneyFromDecimal(decimal.NewFromFloat(req.SpendThreshold))
	existing.IsDefault = req.IsDefault
	existing.SortOrder = req.SortOrder
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}

	if err := h.MemberLevelService.UpdateLevel(existing); err != nil {
		switch {
		case errors.Is(err, service.ErrMemberLevelSlugExists):
			shared.RespondError(c, response.CodeBadRequest, "error.member_level_slug_exists", nil)
		case errors.Is(err, service.ErrMemberLevelNotFound):
			shared.RespondError(c, response.CodeNotFound, "error.member_level_not_found", nil)
		default:
			shared.RespondError(c, response.CodeInternal, "error.member_level_update_failed", err)
		}
		return
	}

	response.Success(c, existing)
}

// DeleteMemberLevel ??????
func (h *Handler) DeleteMemberLevel(c *gin.Context) {
	levelID, err := shared.ParseParamUint(c, "id")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}

	if err := h.MemberLevelService.DeleteLevel(levelID); err != nil {
		switch {
		case errors.Is(err, service.ErrMemberLevelNotFound):
			shared.RespondError(c, response.CodeNotFound, "error.member_level_not_found", nil)
		case errors.Is(err, service.ErrMemberLevelDeleteDefault):
			shared.RespondError(c, response.CodeBadRequest, "error.member_level_cannot_delete_default", nil)
		default:
			shared.RespondError(c, response.CodeInternal, "error.member_level_delete_failed", err)
		}
		return
	}

	response.Success(c, gin.H{"deleted": true})
}

// GetMemberLevelPrices ??????????
func (h *Handler) GetMemberLevelPrices(c *gin.Context) {
	productID, err := shared.ParseQueryUint(c.Query("product_id"), false)
	if err != nil || productID == 0 {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}

	prices, err := h.MemberLevelService.GetLevelPricesByProduct(productID)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.member_level_price_fetch_failed", err)
		return
	}

	response.Success(c, prices)
}

// BatchUpsertMemberLevelPricesRequest ?????????
type BatchUpsertMemberLevelPricesRequest struct {
	Prices []MemberLevelPriceInput `json:"prices" binding:"required"`
}

// MemberLevelPriceInput ?????
type MemberLevelPriceInput struct {
	MemberLevelID uint    `json:"member_level_id" binding:"required"`
	ProductID     uint    `json:"product_id" binding:"required"`
	SKUID         uint    `json:"sku_id"`
	PriceAmount   float64 `json:"price_amount" binding:"required"`
}

// BatchUpsertMemberLevelPrices ???????
func (h *Handler) BatchUpsertMemberLevelPrices(c *gin.Context) {
	var req BatchUpsertMemberLevelPricesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}

	prices := make([]models.MemberLevelPrice, 0, len(req.Prices))
	for _, p := range req.Prices {
		prices = append(prices, models.MemberLevelPrice{
			MemberLevelID: p.MemberLevelID,
			ProductID:     p.ProductID,
			SKUID:         p.SKUID,
			PriceAmount:   models.NewMoneyFromDecimal(decimal.NewFromFloat(p.PriceAmount)),
		})
	}

	if err := h.MemberLevelService.BatchUpsertLevelPrices(prices); err != nil {
		shared.RespondError(c, response.CodeInternal, "error.member_level_price_save_failed", err)
		return
	}

	response.Success(c, gin.H{"saved": true})
}

// DeleteMemberLevelPrice ?????
func (h *Handler) DeleteMemberLevelPrice(c *gin.Context) {
	priceID, err := shared.ParseParamUint(c, "id")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}

	if err := h.MemberLevelService.DeleteLevelPrice(priceID); err != nil {
		shared.RespondError(c, response.CodeInternal, "error.member_level_price_delete_failed", err)
		return
	}

	response.Success(c, gin.H{"deleted": true})
}

// SetUserMemberLevelRequest ??????????
type SetUserMemberLevelRequest struct {
	MemberLevelID uint `json:"member_level_id"`
}

// SetUserMemberLevel ????????
func (h *Handler) SetUserMemberLevel(c *gin.Context) {
	userID, err := shared.ParseParamUint(c, "id")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}

	var req SetUserMemberLevelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}

	if err := h.MemberLevelService.SetUserLevel(userID, req.MemberLevelID); err != nil {
		switch {
		case errors.Is(err, service.ErrMemberLevelNotFound):
			shared.RespondError(c, response.CodeNotFound, "error.member_level_not_found", nil)
		default:
			shared.RespondError(c, response.CodeInternal, "error.user_member_level_update_failed", err)
		}
		return
	}

	response.Success(c, gin.H{"updated": true})
}

// BackfillMemberLevels POST /admin/member-levels/backfill - ????????????????????
func (h *Handler) BackfillMemberLevels(c *gin.Context) {
	affected, err := h.MemberLevelService.BackfillDefaultLevel()
	if err != nil {
		switch {
		case errors.Is(err, service.ErrMemberLevelNotFound):
			shared.RespondError(c, response.CodeBadRequest, "error.member_level_no_default", nil)
		default:
			shared.RespondError(c, response.CodeInternal, "error.member_level_backfill_failed", err)
		}
		return
	}
	response.Success(c, gin.H{"affected": affected})
}
