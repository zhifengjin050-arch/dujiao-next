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

// CreatePromotionRequest ???????
type CreatePromotionRequest struct {
	Name       string  `json:"name" binding:"required"`
	Type       string  `json:"type" binding:"required"`
	ScopeRefID uint    `json:"scope_ref_id" binding:"required"`
	Value      float64 `json:"value" binding:"required"`
	MinAmount  float64 `json:"min_amount"`
	StartsAt   string  `json:"starts_at"`
	EndsAt     string  `json:"ends_at"`
	IsActive   *bool   `json:"is_active"`
}

// CreatePromotion ?????
func (h *Handler) CreatePromotion(c *gin.Context) {
	var req CreatePromotionRequest
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

	promotion, err := h.PromotionAdminService.Create(service.CreatePromotionInput{
		Name:       req.Name,
		Type:       req.Type,
		ScopeRefID: req.ScopeRefID,
		Value:      models.NewMoneyFromDecimal(decimal.NewFromFloat(req.Value)),
		MinAmount:  models.NewMoneyFromDecimal(decimal.NewFromFloat(req.MinAmount)),
		StartsAt:   startsAt,
		EndsAt:     endsAt,
		IsActive:   req.IsActive,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPromotionInvalid):
			shared.RespondError(c, response.CodeBadRequest, "error.promotion_invalid", nil)
		default:
			shared.RespondError(c, response.CodeInternal, "error.promotion_create_failed", err)
		}
		return
	}

	response.Success(c, promotion)
}

// UpdatePromotion ?????
func (h *Handler) UpdatePromotion(c *gin.Context) {
	promotionID, err := shared.ParseParamUint(c, "id")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}
	var req CreatePromotionRequest
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

	promotion, err := h.PromotionAdminService.Update(promotionID, service.UpdatePromotionInput{
		Name:       req.Name,
		Type:       req.Type,
		ScopeRefID: req.ScopeRefID,
		Value:      models.NewMoneyFromDecimal(decimal.NewFromFloat(req.Value)),
		MinAmount:  models.NewMoneyFromDecimal(decimal.NewFromFloat(req.MinAmount)),
		StartsAt:   startsAt,
		EndsAt:     endsAt,
		IsActive:   req.IsActive,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPromotionNotFound):
			shared.RespondError(c, response.CodeNotFound, "error.promotion_not_found", nil)
		case errors.Is(err, service.ErrPromotionInvalid):
			shared.RespondError(c, response.CodeBadRequest, "error.promotion_invalid", nil)
		default:
			shared.RespondError(c, response.CodeInternal, "error.promotion_update_failed", err)
		}
		return
	}

	response.Success(c, promotion)
}

// DeletePromotion ?????
func (h *Handler) DeletePromotion(c *gin.Context) {
	promotionID, err := shared.ParseParamUint(c, "id")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}
	if err := h.PromotionAdminService.Delete(promotionID); err != nil {
		switch {
		case errors.Is(err, service.ErrPromotionNotFound):
			shared.RespondError(c, response.CodeNotFound, "error.promotion_not_found", nil)
		case errors.Is(err, service.ErrPromotionInvalid):
			shared.RespondError(c, response.CodeBadRequest, "error.promotion_invalid", nil)
		default:
			shared.RespondError(c, response.CodeInternal, "error.promotion_delete_failed", err)
		}
		return
	}
	response.Success(c, gin.H{
		"deleted": true,
	})
}

// GetAdminPromotions ???????
func (h *Handler) GetAdminPromotions(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	page, pageSize = shared.NormalizePagination(page, pageSize)

	id, err := shared.ParseQueryUint(c.Query("id"), true)
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}

	scopeRefID, _ := shared.ParseQueryUint(c.Query("scope_ref_id"), false)

	var isActive *bool
	if raw := c.Query("is_active"); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
			return
		}
		isActive = &parsed
	}

	promotions, total, err := h.PromotionAdminService.List(repository.PromotionListFilter{
		ID:         id,
		ScopeRefID: scopeRefID,
		IsActive:   isActive,
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.promotion_fetch_failed", err)
		return
	}

	pagination := response.BuildPagination(page, pageSize, total)
	response.SuccessWithPage(c, promotions, pagination)
}
