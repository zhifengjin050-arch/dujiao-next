package public

import (
	"errors"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/dto"
	"github.com/dujiao-next/internal/http/handlers/shared"
	"github.com/dujiao-next/internal/http/response"
	"github.com/dujiao-next/internal/service"

	"github.com/gin-gonic/gin"
)

// CartItemRequest ??????
type CartItemRequest struct {
	ProductID       uint   `json:"product_id" binding:"required"`
	SKUID           uint   `json:"sku_id"`
	Quantity        int    `json:"quantity" binding:"required"`
	FulfillmentType string `json:"fulfillment_type"`
}

// GetCart ?????
func (h *Handler) GetCart(c *gin.Context) {
	uid, ok := shared.GetUserID(c)
	if !ok {
		return
	}

	items, err := h.CartService.ListByUser(uid)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidOrderItem):
			shared.RespondError(c, response.CodeBadRequest, "error.order_item_invalid", nil)
		case errors.Is(err, service.ErrProductNotAvailable):
			shared.RespondError(c, response.CodeBadRequest, "error.product_not_available", nil)
		case errors.Is(err, service.ErrFulfillmentInvalid):
			shared.RespondError(c, response.CodeBadRequest, "error.fulfillment_invalid", nil)
		case errors.Is(err, service.ErrPromotionInvalid):
			shared.RespondError(c, response.CodeBadRequest, "error.promotion_invalid", nil)
		default:
			shared.RespondError(c, response.CodeInternal, "error.order_fetch_failed", err)
		}
		return
	}

	respItems := make([]dto.CartItemResp, 0, len(items))
	for _, item := range items {
		if item.Product == nil {
			continue
		}
		productFT := item.Product.FulfillmentType
		if productFT == constants.FulfillmentTypeUpstream {
			productFT = constants.FulfillmentTypeManual
		}
		cartFT := item.FulfillmentType
		if cartFT == constants.FulfillmentTypeUpstream {
			cartFT = constants.FulfillmentTypeManual
		}
		product := dto.CartProductResp{
			Slug:                item.Product.Slug,
			Title:               item.Product.TitleJSON,
			PriceAmount:         item.Product.PriceAmount,
			Images:              item.Product.Images,
			Tags:                item.Product.Tags,
			PurchaseType:        item.Product.PurchaseType,
			MaxPurchaseQuantity: item.Product.MaxPurchaseQuantity,
			FulfillmentType:     productFT,
			IsActive:            item.Product.IsActive,
		}
		respItems = append(respItems, dto.CartItemResp{
			ProductID:       item.ProductID,
			SKUID:           item.SKUID,
			Quantity:        item.Quantity,
			FulfillmentType: cartFT,
			UnitPrice:       item.UnitPrice,
			OriginalPrice:   item.OriginalPrice,
			Currency:        item.Currency,
			Product:         product,
		})
	}

	response.Success(c, gin.H{"items": respItems})
}

// UpsertCartItem ??/??????
func (h *Handler) UpsertCartItem(c *gin.Context) {
	uid, ok := shared.GetUserID(c)
	if !ok {
		return
	}
	var req CartItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}
	if req.Quantity <= 0 {
		if err := h.CartService.RemoveItem(uid, req.ProductID, req.SKUID); err != nil {
			shared.RespondError(c, response.CodeInternal, "error.order_update_failed", err)
			return
		}
		response.Success(c, gin.H{"updated": true})
		return
	}
	if err := h.CartService.UpsertItem(service.UpsertCartItemInput{
		UserID:          uid,
		ProductID:       req.ProductID,
		SKUID:           req.SKUID,
		Quantity:        req.Quantity,
		FulfillmentType: req.FulfillmentType,
	}); err != nil {
		switch {
		case errors.Is(err, service.ErrProductSKURequired):
			shared.RespondError(c, response.CodeBadRequest, "error.order_item_invalid", nil)
		case errors.Is(err, service.ErrProductSKUInvalid):
			shared.RespondError(c, response.CodeBadRequest, "error.order_item_invalid", nil)
		case errors.Is(err, service.ErrInvalidOrderItem):
			shared.RespondError(c, response.CodeBadRequest, "error.order_item_invalid", nil)
		case errors.Is(err, service.ErrProductMaxPurchaseExceeded):
			shared.RespondError(c, response.CodeBadRequest, "error.product_max_purchase_exceeded", nil)
		case errors.Is(err, service.ErrProductNotAvailable):
			shared.RespondError(c, response.CodeBadRequest, "error.product_not_available", nil)
		case errors.Is(err, service.ErrManualStockInsufficient):
			shared.RespondError(c, response.CodeBadRequest, "error.manual_stock_insufficient", nil)
		case errors.Is(err, service.ErrFulfillmentInvalid):
			shared.RespondError(c, response.CodeBadRequest, "error.fulfillment_invalid", nil)
		default:
			shared.RespondError(c, response.CodeInternal, "error.order_update_failed", err)
		}
		return
	}
	response.Success(c, gin.H{"updated": true})
}

// DeleteCartItem ??????
func (h *Handler) DeleteCartItem(c *gin.Context) {
	uid, ok := shared.GetUserID(c)
	if !ok {
		return
	}
	productID, err := shared.ParseParamUint(c, "product_id")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.order_item_invalid", nil)
		return
	}
	skuID, err := shared.ParseQueryUint(c.DefaultQuery("sku_id", "0"), false)
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.order_item_invalid", nil)
		return
	}
	if err := h.CartService.RemoveItem(uid, productID, skuID); err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidOrderItem):
			shared.RespondError(c, response.CodeBadRequest, "error.order_item_invalid", nil)
		default:
			shared.RespondError(c, response.CodeInternal, "error.order_update_failed", err)
		}
		return
	}
	response.Success(c, gin.H{"deleted": true})
}
