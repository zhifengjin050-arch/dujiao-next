package channel

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/http/response"
	"github.com/dujiao-next/internal/i18n"
	"github.com/dujiao-next/internal/logger"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/repository"
	"github.com/dujiao-next/internal/service"

	"github.com/gin-gonic/gin"
)

type orderListItem struct {
	OrderID          uint    `json:"order_id"`
	OrderNo          string  `json:"order_no"`
	Status           string  `json:"status"`
	Currency         string  `json:"currency"`
	TotalAmount      string  `json:"total_amount"`
	PaidAmount       string  `json:"paid_amount"`
	WalletPaidAmount string  `json:"wallet_paid_amount"`
	OnlinePaidAmount string  `json:"online_paid_amount"`
	ProductTitle     string  `json:"product_title"`
	ItemCount        int     `json:"item_count"`
	ExpiresAt        *string `json:"expires_at,omitempty"`
	CreatedAt        string  `json:"created_at"`
}

type channelOrderItemRequest struct {
	ProductID       uint   `json:"product_id"`
	SKUID           uint   `json:"sku_id"`
	Quantity        int    `json:"quantity"`
	FulfillmentType string `json:"fulfillment_type"`
}

type previewOrderRequest struct {
	ChannelUserID  string                    `json:"channel_user_id"`
	TelegramUserID string                    `json:"telegram_user_id"`
	Username       string                    `json:"username"`
	TelegramUser   string                    `json:"telegram_username"`
	FirstName      string                    `json:"first_name"`
	LastName       string                    `json:"last_name"`
	AvatarURL      string                    `json:"avatar_url"`
	Locale         string                    `json:"locale"`
	Items          []channelOrderItemRequest `json:"items"`
	CouponCode     string                    `json:"coupon_code"`
	AffiliateCode  string                    `json:"affiliate_code"`
	AffiliateKey   string                    `json:"affiliate_visitor_key"`
	ManualFormData map[string]models.JSON    `json:"manual_form_data"`
}

type createOrderRequest struct {
	ChannelUserID  string                    `json:"channel_user_id"`
	TelegramUserID string                    `json:"telegram_user_id"`
	Username       string                    `json:"username"`
	TelegramUser   string                    `json:"telegram_username"`
	FirstName      string                    `json:"first_name"`
	LastName       string                    `json:"last_name"`
	AvatarURL      string                    `json:"avatar_url"`
	Locale         string                    `json:"locale"`
	Items          []channelOrderItemRequest `json:"items"`
	ProductID      uint                      `json:"product_id"`
	SKUID          uint                      `json:"sku_id"`
	Quantity       int                       `json:"quantity"`
	CouponCode     string                    `json:"coupon_code"`
	AffiliateCode  string                    `json:"affiliate_code"`
	AffiliateKey   string                    `json:"affiliate_visitor_key"`
	ManualFormData map[string]models.JSON    `json:"manual_form_data"`
}

type createPaymentRequest struct {
	ChannelUserID  string `json:"channel_user_id"`
	TelegramUserID string `json:"telegram_user_id"`
	OrderID        uint   `json:"order_id" binding:"required"`
	ChannelID      uint   `json:"channel_id"`
	UseBalance     bool   `json:"use_balance"`
}

type latestPaymentQuery struct {
	OrderID uint `form:"order_id" binding:"required"`
}

type cancelOrderRequest struct {
	ChannelUserID  string `json:"channel_user_id"`
	TelegramUserID string `json:"telegram_user_id"`
	Reason         string `json:"reason"`
}

// PreviewOrder POST /api/v1/channel/orders/preview
func (h *Handler) PreviewOrder(c *gin.Context) {
	var req previewOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondChannelBindError(c, err)
		return
	}

	items, err := buildChannelOrderItems(req.Items, 0, 0, 0)
	if err != nil {
		respondChannelError(c, 400, response.CodeBadRequest, "validation_error", "error.bad_request", nil)
		return
	}

	userID, err := h.provisionTelegramChannelUserID(telegramChannelIdentityInput(
		req.ChannelUserID,
		req.TelegramUserID,
		req.Username,
		req.TelegramUser,
		req.FirstName,
		req.LastName,
		req.AvatarURL,
	))
	if err != nil {
		logger.Errorw("channel_order_preview_resolve_user", "channel_user_id", channelUserIDValue(req.ChannelUserID, req.TelegramUserID), "error", err)
		respondChannelIdentityServiceError(c, err)
		return
	}

	preview, err := h.OrderService.PreviewOrder(service.CreateOrderInput{
		UserID:              userID,
		Items:               items,
		CouponCode:          req.CouponCode,
		AffiliateCode:       req.AffiliateCode,
		AffiliateVisitorKey: req.AffiliateKey,
		ClientIP:            c.ClientIP(),
		ManualFormData:      req.ManualFormData,
	})
	if err != nil {
		logger.Errorw("channel_order_preview", "user_id", userID, "error", err)
		respondChannelOrderPreviewError(c, err)
		return
	}

	locale := channelLocaleValue(c, req.Locale)
	respondChannelSuccess(c, buildChannelOrderPreviewResponse(preview, locale))
}

// CreateOrder POST /api/v1/channel/orders
func (h *Handler) CreateOrder(c *gin.Context) {
	var req createOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondChannelBindError(c, err)
		return
	}

	items, err := buildChannelOrderItems(req.Items, req.ProductID, req.SKUID, req.Quantity)
	if err != nil {
		respondChannelError(c, 400, response.CodeBadRequest, "validation_error", "error.bad_request", nil)
		return
	}

	userID, err := h.provisionTelegramChannelUserID(telegramChannelIdentityInput(
		req.ChannelUserID,
		req.TelegramUserID,
		req.Username,
		req.TelegramUser,
		req.FirstName,
		req.LastName,
		req.AvatarURL,
	))
	if err != nil {
		logger.Errorw("channel_order_resolve_user", "channel_user_id", channelUserIDValue(req.ChannelUserID, req.TelegramUserID), "error", err)
		respondChannelIdentityServiceError(c, err)
		return
	}

	order, err := h.OrderService.CreateOrder(service.CreateOrderInput{
		UserID:              userID,
		Items:               items,
		CouponCode:          req.CouponCode,
		AffiliateCode:       req.AffiliateCode,
		AffiliateVisitorKey: req.AffiliateKey,
		ClientIP:            c.ClientIP(),
		ManualFormData:      req.ManualFormData,
		SkipIPRiskControl:   true, // Bot ??? IP ??,?? IP ????????
	})
	if err != nil {
		logger.Errorw("channel_order_create", "user_id", userID, "error", err)
		respondChannelOrderCreateError(c, err)
		return
	}

	respondChannelSuccess(c, gin.H{
		"order_id":           order.ID,
		"order_no":           order.OrderNo,
		"status":             order.Status,
		"fulfillment_type":   channelOrderFulfillmentType(order),
		"currency":           order.Currency,
		"item_count":         len(order.Items),
		"original_amount":    order.OriginalAmount.StringFixed(2),
		"coupon_discount":    order.DiscountAmount.StringFixed(2),
		"promotion_discount": order.PromotionDiscountAmount.StringFixed(2),
		"total_amount":       order.TotalAmount.StringFixed(2),
		"wallet_paid_amount": order.WalletPaidAmount.StringFixed(2),
		"online_paid_amount": order.OnlinePaidAmount.StringFixed(2),
		"paid_amount":        channelOrderPaidAmount(order),
		"refunded_amount":    order.RefundedAmount.StringFixed(2),
		"expires_at":         order.ExpiresAt,
		"created_at":         order.CreatedAt,
	})
}

// GetPaymentChannels GET /api/v1/channel/payment-channels
// ??????:
//   - order_no: ?????????????????
//   - context: "recharge" ?????????????
func (h *Handler) GetPaymentChannels(c *gin.Context) {
	contextParam := strings.TrimSpace(c.Query("context"))
	orderNo := strings.TrimSpace(c.Query("order_no"))

	var channels []models.PaymentChannel
	var err error

	if contextParam == "recharge" {
		// ??????
		channels, err = h.PaymentService.GetWalletRechargeChannels()
	} else if orderNo != "" {
		// ?????????
		channelUserID := channelUserIDFromQuery(c)
		if channelUserID == "" {
			// ?? channel_user_id,??????,??????
			channels, _, err = h.PaymentService.ListChannels(repository.PaymentChannelListFilter{
				ActiveOnly: true,
				Page:       1,
				PageSize:   50,
			})
		} else {
			userID, resolveErr := h.provisionTelegramChannelUserID(service.TelegramChannelIdentityInput{ChannelUserID: channelUserID})
			if resolveErr != nil {
				logger.Errorw("channel_payment_channels_resolve_user", "channel_user_id", channelUserID, "error", resolveErr)
				channels, _, err = h.PaymentService.ListChannels(repository.PaymentChannelListFilter{
					ActiveOnly: true,
					Page:       1,
					PageSize:   50,
				})
			} else {
				order, orderErr := h.OrderService.GetOrderByUserOrderNo(orderNo, userID)
				if orderErr != nil || order == nil {
					// ?????,??????
					channels, _, err = h.PaymentService.ListChannels(repository.PaymentChannelListFilter{
						ActiveOnly: true,
						Page:       1,
						PageSize:   50,
					})
				} else {
					allItems := order.Items
					for _, child := range order.Children {
						allItems = append(allItems, child.Items...)
					}
					productIDSet := make(map[uint]struct{})
					for _, item := range allItems {
						if item.ProductID > 0 {
							productIDSet[item.ProductID] = struct{}{}
						}
					}
					productIDs := make([]uint, 0, len(productIDSet))
					for id := range productIDSet {
						productIDs = append(productIDs, id)
					}
					channels, err = h.PaymentService.GetAllowedChannelsForProducts(productIDs)
				}
			}
		}
	} else {
		// ??:??????
		channels, _, err = h.PaymentService.ListChannels(repository.PaymentChannelListFilter{
			ActiveOnly: true,
			Page:       1,
			PageSize:   50,
		})
	}

	if err != nil {
		logger.Errorw("channel_order_list_payment_channels", "error", err)
		respondChannelError(c, 500, 500, "internal_error", "error.internal_error", err)
		return
	}

	type channelItem struct {
		ID              uint   `json:"id"`
		Name            string `json:"name"`
		ProviderType    string `json:"provider_type"`
		ChannelType     string `json:"channel_type"`
		InteractionMode string `json:"interaction_mode"`
		FeeRate         string `json:"fee_rate"`
		FixedFee        string `json:"fixed_fee"`
	}

	items := make([]channelItem, 0, len(channels))
	for _, ch := range channels {
		if ch.ProviderType == "balance" || ch.ProviderType == "wallet" {
			continue
		}
		items = append(items, channelItem{
			ID:              ch.ID,
			Name:            ch.Name,
			ProviderType:    ch.ProviderType,
			ChannelType:     ch.ChannelType,
			InteractionMode: ch.InteractionMode,
			FeeRate:         ch.FeeRate.StringFixed(2),
			FixedFee:        ch.FixedFee.StringFixed(2),
		})
	}

	resp := gin.H{"items": items}
	if h.SettingService != nil && h.SettingService.GetWalletOnlyPayment() {
		resp["wallet_only_payment"] = true
	}
	respondChannelSuccess(c, resp)
}

// GetLatestPayment GET /api/v1/channel/payments/latest
func (h *Handler) GetLatestPayment(c *gin.Context) {
	var query latestPaymentQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		respondChannelBindError(c, err)
		return
	}

	channelUserID := channelUserIDFromQuery(c)
	if channelUserID == "" {
		respondChannelError(c, 400, response.CodeBadRequest, "validation_error", "error.bad_request", nil)
		return
	}

	userID, err := h.provisionTelegramChannelUserID(service.TelegramChannelIdentityInput{ChannelUserID: channelUserID})
	if err != nil {
		logger.Errorw("channel_payment_latest_resolve_user", "channel_user_id", channelUserID, "error", err)
		respondChannelIdentityServiceError(c, err)
		return
	}

	order, err := h.OrderService.GetOrderByUser(query.OrderID, userID)
	if err != nil {
		if errors.Is(err, service.ErrOrderNotFound) {
			respondChannelError(c, 404, response.CodeNotFound, "order_not_found", "error.order_not_found", nil)
			return
		}
		respondChannelError(c, 500, response.CodeInternal, "internal_error", "error.order_fetch_failed", err)
		return
	}
	if order == nil {
		respondChannelError(c, 404, response.CodeNotFound, "order_not_found", "error.order_not_found", nil)
		return
	}
	if order.ParentID != nil {
		respondChannelError(c, 400, response.CodeBadRequest, "payment_invalid", "error.payment_invalid", nil)
		return
	}
	if order.Status != constants.OrderStatusPendingPayment {
		respondChannelError(c, 400, response.CodeBadRequest, "order_status_invalid", "error.order_status_invalid", nil)
		return
	}
	if order.ExpiresAt != nil && !order.ExpiresAt.After(time.Now()) {
		respondChannelError(c, 400, response.CodeBadRequest, "order_status_invalid", "error.order_status_invalid", nil)
		return
	}

	payment, err := h.PaymentRepo.GetLatestPendingByOrder(order.ID, time.Now())
	if err != nil {
		respondChannelError(c, 500, response.CodeInternal, "internal_error", "error.payment_fetch_failed", err)
		return
	}
	if payment == nil {
		respondChannelError(c, 404, response.CodeNotFound, "payment_not_found", "error.payment_not_found", nil)
		return
	}

	respondChannelSuccess(c, buildChannelPaymentResponse(order, payment))
}

// CreatePayment POST /api/v1/channel/payments
func (h *Handler) CreatePayment(c *gin.Context) {
	var req createPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondChannelBindError(c, err)
		return
	}

	channelUserID := channelUserIDValue(req.ChannelUserID, req.TelegramUserID)
	if channelUserID == "" {
		respondChannelError(c, 400, response.CodeBadRequest, "validation_error", "error.bad_request", nil)
		return
	}

	userID, err := h.provisionTelegramChannelUserID(service.TelegramChannelIdentityInput{ChannelUserID: channelUserID})
	if err != nil {
		logger.Errorw("channel_payment_resolve_user", "channel_user_id", channelUserID, "error", err)
		respondChannelIdentityServiceError(c, err)
		return
	}

	if _, err := h.OrderService.GetOrderByUser(req.OrderID, userID); err != nil {
		if errors.Is(err, service.ErrOrderNotFound) {
			respondChannelError(c, 404, response.CodeNotFound, "order_not_found", "error.order_not_found", nil)
			return
		}
		respondChannelError(c, 500, response.CodeInternal, "internal_error", "error.internal_error", err)
		return
	}

	result, err := h.PaymentService.CreatePayment(service.CreatePaymentInput{
		OrderID:    req.OrderID,
		ChannelID:  req.ChannelID,
		UseBalance: req.UseBalance,
		ClientIP:   c.ClientIP(),
		Context:    c.Request.Context(),
	})
	if err != nil {
		logger.Errorw("channel_order_create_payment", "order_id", req.OrderID, "error", err)
		respondChannelPaymentCreateError(c, err)
		return
	}

	resp := gin.H{
		"order_paid":         result.OrderPaid,
		"wallet_paid_amount": result.WalletPaidAmount.StringFixed(2),
		"online_pay_amount":  result.OnlinePayAmount.StringFixed(2),
	}
	if result.Payment != nil {
		for key, value := range buildChannelPaymentResponse(nil, result.Payment) {
			resp[key] = value
		}
	}
	if result.Channel != nil {
		resp["channel_name"] = result.Channel.Name
	}

	respondChannelSuccess(c, resp)
}

// GetOrderStatus GET /api/v1/channel/orders/:id
func (h *Handler) GetOrderStatus(c *gin.Context) {
	orderID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		respondChannelError(c, 400, response.CodeBadRequest, "validation_error", "error.bad_request", nil)
		return
	}

	channelUserID := channelUserIDFromQuery(c)
	if channelUserID == "" {
		respondChannelError(c, 400, response.CodeBadRequest, "validation_error", "error.bad_request", nil)
		return
	}

	userID, err := h.provisionTelegramChannelUserID(service.TelegramChannelIdentityInput{ChannelUserID: channelUserID})
	if err != nil {
		logger.Errorw("channel_order_status_resolve_user", "channel_user_id", channelUserID, "error", err)
		respondChannelIdentityServiceError(c, err)
		return
	}

	order, err := h.OrderService.GetOrderByUser(uint(orderID), userID)
	if err != nil {
		if errors.Is(err, service.ErrOrderNotFound) {
			respondChannelError(c, 404, response.CodeNotFound, "order_not_found", "error.order_not_found", nil)
			return
		}
		respondChannelError(c, 500, response.CodeInternal, "internal_error", "error.internal_error", err)
		return
	}
	if order == nil {
		respondChannelError(c, 404, response.CodeNotFound, "order_not_found", "error.order_not_found", nil)
		return
	}

	order.MaskUpstreamFulfillmentType()
	order.StripCostPrice()
	respondChannelSuccess(c, buildChannelOrderDetailResponse(order, channelLocaleValue(c, c.Query("locale"))))
}

// GetOrderByOrderNo GET /api/v1/channel/orders/by-order-no/:order_no
func (h *Handler) GetOrderByOrderNo(c *gin.Context) {
	orderNo := strings.TrimSpace(c.Param("order_no"))
	if orderNo == "" {
		respondChannelError(c, 400, response.CodeBadRequest, "validation_error", "error.bad_request", nil)
		return
	}

	channelUserID := channelUserIDFromQuery(c)
	if channelUserID == "" {
		respondChannelError(c, 400, response.CodeBadRequest, "validation_error", "error.bad_request", nil)
		return
	}

	userID, err := h.provisionTelegramChannelUserID(service.TelegramChannelIdentityInput{ChannelUserID: channelUserID})
	if err != nil {
		logger.Errorw("channel_order_by_no_resolve_user", "channel_user_id", channelUserID, "error", err)
		respondChannelIdentityServiceError(c, err)
		return
	}

	order, err := h.OrderService.GetOrderByUserOrderNo(orderNo, userID)
	if err != nil {
		if errors.Is(err, service.ErrOrderNotFound) {
			respondChannelError(c, 404, response.CodeNotFound, "order_not_found", "error.order_not_found", nil)
			return
		}
		respondChannelError(c, 500, response.CodeInternal, "internal_error", "error.order_fetch_failed", err)
		return
	}
	if order == nil {
		respondChannelError(c, 404, response.CodeNotFound, "order_not_found", "error.order_not_found", nil)
		return
	}

	order.MaskUpstreamFulfillmentType()
	order.StripCostPrice()
	respondChannelSuccess(c, buildChannelOrderDetailResponse(order, channelLocaleValue(c, c.Query("locale"))))
}

// CancelOrder POST /api/v1/channel/orders/:id/cancel
func (h *Handler) CancelOrder(c *gin.Context) {
	orderID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		respondChannelError(c, 400, response.CodeBadRequest, "validation_error", "error.bad_request", nil)
		return
	}

	var req cancelOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondChannelBindError(c, err)
		return
	}

	channelUserID := channelUserIDValue(req.ChannelUserID, req.TelegramUserID)
	if channelUserID == "" {
		respondChannelError(c, 400, response.CodeBadRequest, "validation_error", "error.bad_request", nil)
		return
	}

	userID, err := h.provisionTelegramChannelUserID(service.TelegramChannelIdentityInput{ChannelUserID: channelUserID})
	if err != nil {
		logger.Errorw("channel_order_cancel_resolve_user", "channel_user_id", channelUserID, "error", err)
		respondChannelIdentityServiceError(c, err)
		return
	}

	order, err := h.OrderService.CancelOrder(uint(orderID), userID)
	if err != nil {
		logger.Errorw("channel_order_cancel", "order_id", orderID, "error", err)
		if errors.Is(err, service.ErrOrderNotFound) {
			respondChannelError(c, 404, response.CodeNotFound, "order_not_found", "error.order_not_found", nil)
			return
		}
		respondChannelError(c, 400, response.CodeBadRequest, "order_status_invalid", "error.order_status_invalid", err)
		return
	}

	respondChannelSuccess(c, gin.H{
		"order_id":     order.ID,
		"order_no":     order.OrderNo,
		"status":       order.Status,
		"cancelled_at": order.CanceledAt,
	})
}

// ListOrders GET /api/v1/channel/orders
func (h *Handler) ListOrders(c *gin.Context) {
	channelUserID := channelUserIDFromQuery(c)
	if channelUserID == "" {
		respondChannelError(c, 400, response.CodeBadRequest, "validation_error", "error.bad_request", nil)
		return
	}

	userID, err := h.provisionTelegramChannelUserID(service.TelegramChannelIdentityInput{ChannelUserID: channelUserID})
	if err != nil {
		logger.Errorw("channel_order_list_resolve_user", "channel_user_id", channelUserID, "error", err)
		respondChannelIdentityServiceError(c, err)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "5"))
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 20 {
		pageSize = 5
	}
	status := c.Query("status")
	locale := channelLocaleValue(c, c.Query("locale"))

	orders, total, err := h.OrderService.ListOrdersByUser(repository.OrderListFilter{
		Page:     page,
		PageSize: pageSize,
		UserID:   userID,
		Status:   status,
	})
	if err != nil {
		logger.Errorw("channel_order_list", "user_id", userID, "error", err)
		respondChannelError(c, 500, response.CodeInternal, "internal_error", "error.internal_error", err)
		return
	}

	items := make([]orderListItem, 0, len(orders))
	for _, order := range orders {
		productTitle := ""
		if len(order.Items) > 0 {
			productTitle = resolveLocalizedJSON(order.Items[0].TitleJSON, locale, "zh-CN")
		}
		items = append(items, orderListItem{
			OrderID:          order.ID,
			OrderNo:          order.OrderNo,
			Status:           order.Status,
			Currency:         order.Currency,
			TotalAmount:      order.TotalAmount.StringFixed(2),
			PaidAmount:       channelOrderPaidAmount(&order),
			WalletPaidAmount: order.WalletPaidAmount.StringFixed(2),
			OnlinePaidAmount: order.OnlinePaidAmount.StringFixed(2),
			ProductTitle:     productTitle,
			ItemCount:        len(order.Items),
			ExpiresAt:        formatChannelNullableTime(order.ExpiresAt),
			CreatedAt:        order.CreatedAt.Format(time.RFC3339),
		})
	}

	totalPages := (total + int64(pageSize) - 1) / int64(pageSize)
	respondChannelSuccess(c, gin.H{
		"items":       items,
		"page":        page,
		"page_size":   pageSize,
		"total":       total,
		"total_pages": totalPages,
	})
}

// GetPaymentDetail GET /api/v1/channel/payments/:id
func (h *Handler) GetPaymentDetail(c *gin.Context) {
	paymentID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || paymentID == 0 {
		respondChannelError(c, 400, response.CodeBadRequest, "validation_error", "error.bad_request", nil)
		return
	}

	channelUserID := channelUserIDFromQuery(c)
	if channelUserID == "" {
		respondChannelError(c, 400, response.CodeBadRequest, "validation_error", "error.bad_request", nil)
		return
	}

	userID, err := h.provisionTelegramChannelUserID(service.TelegramChannelIdentityInput{ChannelUserID: channelUserID})
	if err != nil {
		logger.Errorw("channel_payment_detail_resolve_user", "channel_user_id", channelUserID, "error", err)
		respondChannelIdentityServiceError(c, err)
		return
	}

	payment, err := h.PaymentRepo.GetByID(uint(paymentID))
	if err != nil {
		respondChannelError(c, 500, response.CodeInternal, "internal_error", "error.payment_fetch_failed", err)
		return
	}
	if payment == nil {
		respondChannelError(c, 404, response.CodeNotFound, "payment_not_found", "error.payment_not_found", nil)
		return
	}

	order, err := h.OrderService.GetOrderByUser(payment.OrderID, userID)
	if err != nil {
		if errors.Is(err, service.ErrOrderNotFound) {
			respondChannelError(c, 404, response.CodeNotFound, "payment_not_found", "error.payment_not_found", nil)
			return
		}
		respondChannelError(c, 500, response.CodeInternal, "internal_error", "error.order_fetch_failed", err)
		return
	}
	if order == nil {
		respondChannelError(c, 404, response.CodeNotFound, "payment_not_found", "error.payment_not_found", nil)
		return
	}

	respondChannelSuccess(c, buildChannelPaymentResponse(order, payment))
}

func buildChannelOrderItems(items []channelOrderItemRequest, legacyProductID, legacySKUID uint, legacyQuantity int) ([]service.CreateOrderItem, error) {
	if len(items) == 0 {
		if legacyProductID == 0 || legacyQuantity <= 0 {
			return nil, service.ErrInvalidOrderItem
		}
		items = []channelOrderItemRequest{{
			ProductID: legacyProductID,
			SKUID:     legacySKUID,
			Quantity:  legacyQuantity,
		}}
	}

	result := make([]service.CreateOrderItem, 0, len(items))
	for _, item := range items {
		if item.ProductID == 0 || item.Quantity <= 0 {
			return nil, service.ErrInvalidOrderItem
		}
		result = append(result, service.CreateOrderItem{
			ProductID:       item.ProductID,
			SKUID:           item.SKUID,
			Quantity:        item.Quantity,
			FulfillmentType: item.FulfillmentType,
		})
	}
	return result, nil
}

func buildChannelOrderPreviewResponse(preview *service.OrderPreview, locale string) gin.H {
	items := make([]gin.H, 0, len(preview.Items))
	for _, item := range preview.Items {
		items = append(items, gin.H{
			"product_id":         item.ProductID,
			"product_title":      resolveLocalizedJSON(item.TitleJSON, locale, "zh-CN"),
			"sku_id":             item.SKUID,
			"sku_name":           channelLocalizedValue(item.SKUSnapshotJSON["spec_values"], locale, "zh-CN"),
			"quantity":           item.Quantity,
			"unit_price":         item.UnitPrice.StringFixed(2),
			"subtotal":           item.TotalPrice.StringFixed(2),
			"coupon_discount":    item.CouponDiscount.StringFixed(2),
			"promotion_discount": item.PromotionDiscount.StringFixed(2),
			"fulfillment_type":   item.FulfillmentType,
		})
	}
	return gin.H{
		"item_count":         len(items),
		"original_amount":    preview.OriginalAmount.StringFixed(2),
		"items":              items,
		"coupon_discount":    preview.DiscountAmount.StringFixed(2),
		"promotion_discount": preview.PromotionDiscountAmount.StringFixed(2),
		"total_amount":       preview.TotalAmount.StringFixed(2),
		"currency":           preview.Currency,
		"valid":              true,
		"validation_errors":  []string{},
	}
}

// joinLocalizedInstructions ?? items ????????(??,? locale ??)?
func joinLocalizedInstructions(items []models.OrderItem, locale string) string {
	if len(items) == 0 {
		return ""
	}
	seen := make(map[string]struct{}, len(items))
	var parts []string
	for _, item := range items {
		text := strings.TrimSpace(resolveLocalizedJSON(item.InstructionsJSON, locale, "zh-CN"))
		if text == "" {
			continue
		}
		if _, ok := seen[text]; ok {
			continue
		}
		seen[text] = struct{}{}
		parts = append(parts, text)
	}
	return strings.Join(parts, "\n\n")
}

func buildChannelOrderDetailResponse(order *models.Order, locale string) gin.H {
	resp := gin.H{
		"order_id":           order.ID,
		"order_no":           order.OrderNo,
		"status":             order.Status,
		"fulfillment_type":   channelOrderFulfillmentType(order),
		"currency":           order.Currency,
		"item_count":         len(order.Items),
		"original_amount":    order.OriginalAmount.StringFixed(2),
		"coupon_discount":    order.DiscountAmount.StringFixed(2),
		"promotion_discount": order.PromotionDiscountAmount.StringFixed(2),
		"total_amount":       order.TotalAmount.StringFixed(2),
		"wallet_paid_amount": order.WalletPaidAmount.StringFixed(2),
		"online_paid_amount": order.OnlinePaidAmount.StringFixed(2),
		"paid_amount":        channelOrderPaidAmount(order),
		"refunded_amount":    order.RefundedAmount.StringFixed(2),
		"expires_at":         order.ExpiresAt,
		"created_at":         order.CreatedAt,
		"updated_at":         order.UpdatedAt,
		"paid_at":            order.PaidAt,
		"cancelled_at":       order.CanceledAt,
	}

	orderPaid := order.PaidAt != nil
	items := make([]gin.H, 0, len(order.Items))
	for _, item := range order.Items {
		instructions := ""
		if orderPaid {
			instructions = resolveLocalizedJSON(item.InstructionsJSON, locale, "zh-CN")
		}
		items = append(items, gin.H{
			"product_id":         item.ProductID,
			"product_title":      resolveLocalizedJSON(item.TitleJSON, locale, "zh-CN"),
			"sku_id":             item.SKUID,
			"sku_name":           channelLocalizedValue(item.SKUSnapshotJSON["spec_values"], locale, "zh-CN"),
			"quantity":           item.Quantity,
			"unit_price":         item.UnitPrice.StringFixed(2),
			"subtotal":           item.TotalPrice.StringFixed(2),
			"coupon_discount":    item.CouponDiscount.StringFixed(2),
			"promotion_discount": item.PromotionDiscount.StringFixed(2),
			"fulfillment_type":   item.FulfillmentType,
			"instructions":       instructions,
		})
	}
	resp["items"] = items

	children := make([]gin.H, 0, len(order.Children))
	for _, child := range order.Children {
		childInstructions := ""
		if orderPaid {
			childInstructions = joinLocalizedInstructions(child.Items, locale)
		}
		childResp := gin.H{
			"order_id": child.ID,
			"order_no": child.OrderNo,
			"status":   child.Status,
		}
		if child.Fulfillment != nil {
			childResp["fulfillment"] = gin.H{
				"status":       child.Fulfillment.Status,
				"type":         child.Fulfillment.Type,
				"payload":      child.Fulfillment.Payload,
				"delivered_at": child.Fulfillment.DeliveredAt,
				"instructions": childInstructions,
			}
		} else {
			childResp["fulfillment"] = nil
		}
		children = append(children, childResp)
	}
	resp["children"] = children

	parentInstructions := ""
	if orderPaid {
		parentInstructions = joinLocalizedInstructions(order.Items, locale)
	}
	if order.Fulfillment != nil {
		resp["fulfillment_status"] = order.Fulfillment.Status
		resp["fulfillment_result"] = order.Fulfillment.Payload
		resp["fulfillment_delivered_at"] = order.Fulfillment.DeliveredAt
		resp["fulfillment_instructions"] = parentInstructions
	} else {
		resp["fulfillment_status"] = ""
		resp["fulfillment_result"] = nil
		resp["fulfillment_delivered_at"] = nil
		resp["fulfillment_instructions"] = ""
	}

	return resp
}

func channelOrderFulfillmentType(order *models.Order) string {
	if order == nil {
		return ""
	}
	if order.Fulfillment != nil {
		if fulfillmentType := strings.TrimSpace(order.Fulfillment.Type); fulfillmentType != "" {
			return fulfillmentType
		}
	}
	for _, item := range order.Items {
		if fulfillmentType := strings.TrimSpace(item.FulfillmentType); fulfillmentType != "" {
			return fulfillmentType
		}
	}
	for _, child := range order.Children {
		if child.Fulfillment != nil {
			if fulfillmentType := strings.TrimSpace(child.Fulfillment.Type); fulfillmentType != "" {
				return fulfillmentType
			}
		}
		for _, item := range child.Items {
			if fulfillmentType := strings.TrimSpace(item.FulfillmentType); fulfillmentType != "" {
				return fulfillmentType
			}
		}
	}
	return ""
}

func buildChannelPaymentResponse(order *models.Order, payment *models.Payment) gin.H {
	resp := gin.H{
		"payment_id":       payment.ID,
		"order_id":         payment.OrderID,
		"channel_id":       payment.ChannelID,
		"status":           payment.Status,
		"provider_type":    payment.ProviderType,
		"channel_type":     payment.ChannelType,
		"interaction_mode": payment.InteractionMode,
		"amount":           payment.Amount.StringFixed(2),
		"fee_rate":         payment.FeeRate.StringFixed(2),
		"fee_amount":       payment.FeeAmount.StringFixed(2),
		"currency":         payment.Currency,
		"pay_url":          payment.PayURL,
		"qr_code":          payment.QRCode,
		"paid_at":          payment.PaidAt,
		"expires_at":       payment.ExpiredAt,
		"callback_at":      payment.CallbackAt,
		"created_at":       payment.CreatedAt,
		"updated_at":       payment.UpdatedAt,
	}
	if order != nil {
		resp["order_no"] = order.OrderNo
		resp["total_amount"] = order.TotalAmount.StringFixed(2)
		resp["wallet_paid_amount"] = order.WalletPaidAmount.StringFixed(2)
		resp["online_paid_amount"] = order.OnlinePaidAmount.StringFixed(2)
		resp["paid_amount"] = channelOrderPaidAmount(order)
	}
	return resp
}

func channelOrderPaidAmount(order *models.Order) string {
	if order == nil {
		return "0.00"
	}
	return models.NewMoneyFromDecimal(order.WalletPaidAmount.Decimal.Add(order.OnlinePaidAmount.Decimal)).StringFixed(2)
}

func formatChannelNullableTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.Format(time.RFC3339)
	return &formatted
}

func channelLocalizedValue(value interface{}, locale, defaultLocale string) string {
	switch typed := value.(type) {
	case models.JSON:
		return resolveLocalizedJSON(typed, locale, defaultLocale)
	case map[string]interface{}:
		return resolveLocalizedJSON(models.JSON(typed), locale, defaultLocale)
	case nil:
		return ""
	default:
		text := strings.TrimSpace(fmt.Sprintf("%v", typed))
		if text == "<nil>" {
			return ""
		}
		return text
	}
}

func channelLocaleValue(c *gin.Context, explicit string) string {
	if value := strings.TrimSpace(explicit); value != "" {
		return value
	}
	return i18n.ResolveLocale(c)
}
