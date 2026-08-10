package channel

import (
	"errors"
	"strconv"
	"strings"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/dto"
	"github.com/dujiao-next/internal/logger"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/repository"
	"github.com/dujiao-next/internal/service"
	"github.com/shopspring/decimal"

	"github.com/gin-gonic/gin"
)

// RedeemGiftCardRequest Channel ???????
type RedeemGiftCardRequest struct {
	ChannelUserID  string `json:"channel_user_id"`
	TelegramUserID string `json:"telegram_user_id"`
	Code           string `json:"code" binding:"required"`
}

// GetWallet GET /api/v1/channel/wallet?telegram_user_id=xxx
func (h *Handler) GetWallet(c *gin.Context) {
	channelUserID := channelUserIDFromQuery(c)
	if channelUserID == "" {
		respondChannelError(c, 400, 400, "validation_error", "error.bad_request", nil)
		return
	}

	userID, err := h.provisionTelegramChannelUserID(service.TelegramChannelIdentityInput{ChannelUserID: channelUserID})
	if err != nil {
		logger.Errorw("channel_wallet_resolve_user", "channel_user_id", channelUserID, "error", err)
		respondChannelIdentityServiceError(c, err)
		return
	}

	account, err := h.WalletService.GetAccount(userID)
	if err != nil {
		logger.Errorw("channel_wallet_get_account", "user_id", userID, "error", err)
		respondChannelError(c, 500, 500, "internal_error", "error.internal_error", err)
		return
	}

	respondChannelSuccess(c, gin.H{
		"balance":  account.Balance.StringFixed(2),
		"currency": "CNY",
	})
}

// GetWalletTransactions GET /api/v1/channel/wallet/transactions?telegram_user_id=xxx&page=1&page_size=5
func (h *Handler) GetWalletTransactions(c *gin.Context) {
	channelUserID := channelUserIDFromQuery(c)
	if channelUserID == "" {
		respondChannelError(c, 400, 400, "validation_error", "error.bad_request", nil)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "5"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 20 {
		pageSize = 5
	}

	userID, err := h.provisionTelegramChannelUserID(service.TelegramChannelIdentityInput{ChannelUserID: channelUserID})
	if err != nil {
		logger.Errorw("channel_wallet_txns_resolve_user", "channel_user_id", channelUserID, "error", err)
		respondChannelIdentityServiceError(c, err)
		return
	}

	txns, total, err := h.WalletService.ListTransactions(repository.WalletTransactionListFilter{
		UserID:   userID,
		Page:     page,
		PageSize: pageSize,
	})
	if err != nil {
		logger.Errorw("channel_wallet_list_txns", "user_id", userID, "error", err)
		respondChannelError(c, 500, 500, "internal_error", "error.internal_error", err)
		return
	}

	type txnItem struct {
		Type         string `json:"type"`
		Direction    string `json:"direction"`
		Amount       string `json:"amount"`
		BalanceAfter string `json:"balance_after"`
		Remark       string `json:"remark"`
		CreatedAt    string `json:"created_at"`
	}

	items := make([]txnItem, 0, len(txns))
	for _, t := range txns {
		items = append(items, txnItem{
			Type:         t.Type,
			Direction:    t.Direction,
			Amount:       t.Amount.StringFixed(2),
			BalanceAfter: t.BalanceAfter.StringFixed(2),
			Remark:       t.Remark,
			CreatedAt:    t.CreatedAt.Format("2006-01-02 15:04"),
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

// RedeemGiftCard POST /api/v1/channel/wallet/gift-card/redeem
func (h *Handler) RedeemGiftCard(c *gin.Context) {
	var req RedeemGiftCardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondChannelBindError(c, err)
		return
	}

	channelUserID := channelUserIDValue(req.ChannelUserID, req.TelegramUserID)
	if channelUserID == "" {
		respondChannelError(c, 400, 400, "validation_error", "error.bad_request", nil)
		return
	}

	userID, err := h.provisionTelegramChannelUserID(service.TelegramChannelIdentityInput{ChannelUserID: channelUserID})
	if err != nil {
		logger.Errorw("channel_wallet_gift_card_resolve_user", "channel_user_id", channelUserID, "error", err)
		respondChannelIdentityServiceError(c, err)
		return
	}

	card, account, txn, err := h.GiftCardService.RedeemGiftCard(service.GiftCardRedeemInput{
		UserID: userID,
		Code:   strings.TrimSpace(req.Code),
	})
	if err != nil {
		logger.Warnw("channel_wallet_gift_card_redeem_failed", "user_id", userID, "channel_user_id", channelUserID, "error", err)
		switch {
		case errors.Is(err, service.ErrGiftCardInvalid):
			respondChannelError(c, 400, 400, "gift_card_invalid", "error.gift_card_invalid", nil)
		case errors.Is(err, service.ErrGiftCardNotFound):
			respondChannelError(c, 404, 404, "gift_card_not_found", "error.gift_card_not_found", nil)
		case errors.Is(err, service.ErrGiftCardExpired):
			respondChannelError(c, 400, 400, "gift_card_expired", "error.gift_card_expired", nil)
		case errors.Is(err, service.ErrGiftCardDisabled):
			respondChannelError(c, 400, 400, "gift_card_disabled", "error.gift_card_disabled", nil)
		case errors.Is(err, service.ErrGiftCardRedeemed):
			respondChannelError(c, 400, 400, "gift_card_redeemed", "error.gift_card_redeemed", nil)
		default:
			respondChannelError(c, 500, 500, "gift_card_redeem_failed", "error.gift_card_redeem_failed", err)
		}
		return
	}

	respondChannelSuccess(c, dto.NewGiftCardRedeemResp(card, account, txn))
}

// CreateWalletRecharge POST /api/v1/channel/wallet/recharge
func (h *Handler) CreateWalletRecharge(c *gin.Context) {
	var req struct {
		ChannelUserID  string `json:"channel_user_id"`
		TelegramUserID string `json:"telegram_user_id"`
		Amount         string `json:"amount" binding:"required"`
		ChannelID      uint   `json:"channel_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		respondChannelBindError(c, err)
		return
	}
	channelUserID := channelUserIDValue(req.ChannelUserID, req.TelegramUserID)
	if channelUserID == "" {
		respondChannelError(c, 400, 400, "validation_error", "error.bad_request", nil)
		return
	}

	userID, err := h.provisionTelegramChannelUserID(service.TelegramChannelIdentityInput{ChannelUserID: channelUserID})
	if err != nil {
		logger.Errorw("channel_wallet_recharge_resolve_user", "channel_user_id", channelUserID, "error", err)
		respondChannelIdentityServiceError(c, err)
		return
	}

	amount, err := decimal.NewFromString(strings.TrimSpace(req.Amount))
	if err != nil {
		respondChannelError(c, 400, 400, "validation_error", "error.bad_request", nil)
		return
	}

	currency, _ := h.SettingService.GetSiteCurrency(constants.SiteCurrencyDefault)

	result, err := h.PaymentService.CreateWalletRechargePayment(service.CreateWalletRechargePaymentInput{
		UserID:    userID,
		ChannelID: req.ChannelID,
		Amount:    models.NewMoneyFromDecimal(amount),
		Currency:  currency,
		ClientIP:  c.ClientIP(),
		Context:   c.Request.Context(),
	})
	if err != nil {
		logger.Errorw("channel_wallet_recharge_create", "user_id", userID, "error", err)
		respondChannelError(c, 400, 400, "payment_create_failed", "error.payment_create_failed", err)
		return
	}

	respondChannelSuccess(c, gin.H{
		"recharge_no": result.Recharge.RechargeNo,
		"payment": gin.H{
			"id":         result.Payment.ID,
			"amount":     result.Payment.Amount.StringFixed(2),
			"fee_amount": result.Payment.FeeAmount.StringFixed(2),
			"currency":   result.Payment.Currency,
			"status":     result.Payment.Status,
			"pay_url":    result.Payment.PayURL,
			"qr_code":    result.Payment.QRCode,
			"expires_at": result.Payment.ExpiredAt,
		},
	})
}
