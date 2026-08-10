package admin

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/dujiao-next/internal/http/handlers/shared"
	"github.com/dujiao-next/internal/http/response"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

// GenerateGiftCardsRequest ???????
type GenerateGiftCardsRequest struct {
	Name      string `json:"name" binding:"required"`
	Quantity  int    `json:"quantity" binding:"required"`
	Amount    string `json:"amount" binding:"required"`
	ExpiresAt string `json:"expires_at"`
}

// UpdateGiftCardRequest ???????
type UpdateGiftCardRequest struct {
	Name      *string `json:"name"`
	Status    *string `json:"status"`
	ExpiresAt *string `json:"expires_at"`
}

// BatchUpdateGiftCardStatusRequest ???????????
type BatchUpdateGiftCardStatusRequest struct {
	IDs    []uint `json:"ids" binding:"required"`
	Status string `json:"status" binding:"required"`
}

// ExportGiftCardRequest ???????
type ExportGiftCardRequest struct {
	IDs    []uint `json:"ids" binding:"required"`
	Format string `json:"format" binding:"required"`
}

type adminGiftCardUser struct {
	ID          uint   `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}

type adminGiftCardItem struct {
	models.GiftCard
	IsExpired    bool               `json:"is_expired"`
	RedeemedUser *adminGiftCardUser `json:"redeemed_user,omitempty"`
}

// GenerateGiftCards ????????
func (h *Handler) GenerateGiftCards(c *gin.Context) {
	adminID, ok := shared.GetAdminID(c)
	if !ok {
		return
	}
	var req GenerateGiftCardsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}
	amount, err := decimal.NewFromString(strings.TrimSpace(req.Amount))
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}
	expiresAt, err := shared.ParseTimeNullable(strings.TrimSpace(req.ExpiresAt))
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}
	batch, created, err := h.GiftCardService.GenerateGiftCards(service.GenerateGiftCardsInput{
		Name:      req.Name,
		Quantity:  req.Quantity,
		Amount:    models.NewMoneyFromDecimal(amount),
		ExpiresAt: expiresAt,
		CreatedBy: &adminID,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrGiftCardInvalid):
			shared.RespondError(c, response.CodeBadRequest, "error.gift_card_invalid", nil)
		default:
			shared.RespondError(c, response.CodeInternal, "error.gift_card_create_failed", err)
		}
		return
	}
	response.Success(c, gin.H{
		"batch":   batch,
		"created": created,
	})
}

// GetGiftCards ???????
func (h *Handler) GetGiftCards(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	page, pageSize = shared.NormalizePagination(page, pageSize)

	status := strings.TrimSpace(strings.ToLower(c.Query("status")))
	code := strings.TrimSpace(c.Query("code"))
	batchNo := strings.TrimSpace(c.Query("batch_no"))

	var redeemedUserID uint
	if rawUserID := strings.TrimSpace(c.Query("redeemed_user_id")); rawUserID != "" {
		parsed, err := shared.ParseQueryUint(rawUserID, true)
		if err != nil {
			shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
			return
		}
		redeemedUserID = parsed
	}

	createdFrom, err := shared.ParseTimeNullable(strings.TrimSpace(c.Query("created_from")))
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}
	createdTo, err := shared.ParseTimeNullable(strings.TrimSpace(c.Query("created_to")))
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}
	redeemedFrom, err := shared.ParseTimeNullable(strings.TrimSpace(c.Query("redeemed_from")))
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}
	redeemedTo, err := shared.ParseTimeNullable(strings.TrimSpace(c.Query("redeemed_to")))
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}
	expiresFrom, err := shared.ParseTimeNullable(strings.TrimSpace(c.Query("expires_from")))
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}
	expiresTo, err := shared.ParseTimeNullable(strings.TrimSpace(c.Query("expires_to")))
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}

	cards, total, err := h.GiftCardService.ListGiftCards(service.GiftCardListInput{
		Code:           code,
		Status:         status,
		BatchNo:        batchNo,
		RedeemedUserID: redeemedUserID,
		CreatedFrom:    createdFrom,
		CreatedTo:      createdTo,
		RedeemedFrom:   redeemedFrom,
		RedeemedTo:     redeemedTo,
		ExpiresFrom:    expiresFrom,
		ExpiresTo:      expiresTo,
		Page:           page,
		PageSize:       pageSize,
	})
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.gift_card_fetch_failed", err)
		return
	}

	userMap, err := h.GiftCardService.ResolveRedeemedUsers(cards)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.gift_card_fetch_failed", err)
		return
	}

	now := time.Now()
	items := make([]adminGiftCardItem, 0, len(cards))
	for _, card := range cards {
		item := adminGiftCardItem{
			GiftCard:  card,
			IsExpired: card.ExpiresAt != nil && card.ExpiresAt.Before(now),
		}
		if card.RedeemedUserID != nil {
			if user, ok := userMap[*card.RedeemedUserID]; ok {
				item.RedeemedUser = &adminGiftCardUser{
					ID:          user.ID,
					Email:       user.Email,
					DisplayName: user.DisplayName,
				}
			}
		}
		items = append(items, item)
	}

	pagination := response.BuildPagination(page, pageSize, total)
	response.SuccessWithPage(c, items, pagination)
}

// UpdateGiftCard ?????
func (h *Handler) UpdateGiftCard(c *gin.Context) {
	id, err := shared.ParseParamUint(c, "id")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", nil)
		return
	}
	var req UpdateGiftCardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}
	var (
		expiresAt      *time.Time
		clearExpiresAt bool
	)
	if req.ExpiresAt != nil {
		if strings.TrimSpace(*req.ExpiresAt) == "" {
			clearExpiresAt = true
		} else {
			parsed, err := shared.ParseTimeNullable(strings.TrimSpace(*req.ExpiresAt))
			if err != nil {
				shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
				return
			}
			expiresAt = parsed
		}
	}
	card, err := h.GiftCardService.UpdateGiftCard(id, service.UpdateGiftCardInput{
		Name:           req.Name,
		Status:         req.Status,
		ExpiresAt:      expiresAt,
		ClearExpiresAt: clearExpiresAt,
	})
	if err != nil {
		switch {
		case errors.Is(err, service.ErrGiftCardNotFound):
			shared.RespondError(c, response.CodeNotFound, "error.gift_card_not_found", nil)
		case errors.Is(err, service.ErrGiftCardInvalid):
			shared.RespondError(c, response.CodeBadRequest, "error.gift_card_invalid", nil)
		default:
			shared.RespondError(c, response.CodeInternal, "error.gift_card_update_failed", err)
		}
		return
	}
	response.Success(c, card)
}

// DeleteGiftCard ?????
func (h *Handler) DeleteGiftCard(c *gin.Context) {
	id, err := shared.ParseParamUint(c, "id")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", nil)
		return
	}
	if err := h.GiftCardService.DeleteGiftCard(id); err != nil {
		switch {
		case errors.Is(err, service.ErrGiftCardNotFound):
			shared.RespondError(c, response.CodeNotFound, "error.gift_card_not_found", nil)
		case errors.Is(err, service.ErrGiftCardInvalid):
			shared.RespondError(c, response.CodeBadRequest, "error.gift_card_invalid", nil)
		default:
			shared.RespondError(c, response.CodeInternal, "error.gift_card_delete_failed", err)
		}
		return
	}
	response.Success(c, gin.H{"deleted": true})
}

// BatchUpdateGiftCardStatus ?????????
func (h *Handler) BatchUpdateGiftCardStatus(c *gin.Context) {
	var req BatchUpdateGiftCardStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}
	affected, err := h.GiftCardService.BatchUpdateStatus(req.IDs, req.Status)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrGiftCardInvalid):
			shared.RespondError(c, response.CodeBadRequest, "error.gift_card_invalid", nil)
		default:
			shared.RespondError(c, response.CodeInternal, "error.gift_card_update_failed", err)
		}
		return
	}
	response.Success(c, gin.H{"affected": affected})
}

// ExportGiftCards ?????
func (h *Handler) ExportGiftCards(c *gin.Context) {
	var req ExportGiftCardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}
	content, contentType, err := h.GiftCardService.ExportGiftCards(req.IDs, req.Format)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrGiftCardNotFound):
			shared.RespondError(c, response.CodeNotFound, "error.gift_card_not_found", nil)
		case errors.Is(err, service.ErrGiftCardInvalid):
			shared.RespondError(c, response.CodeBadRequest, "error.gift_card_invalid", nil)
		default:
			shared.RespondError(c, response.CodeInternal, "error.gift_card_fetch_failed", err)
		}
		return
	}
	filename := fmt.Sprintf("gift_cards_%s.%s", time.Now().Format("20060102_150405"), strings.ToLower(strings.TrimSpace(req.Format)))
	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	c.Data(http.StatusOK, contentType, content)
}
