package public

import (
	"bytes"
	"io"
	"strings"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/http/handlers/shared"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/payment/tokenpay"
	"github.com/dujiao-next/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

func (h *Handler) HandleTokenPayCallback(c *gin.Context) bool {
	log := shared.RequestLog(c)

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return false
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	data, err := tokenpay.ParseCallback(body)
	if err != nil {
		log.Debugw("tokenpay_callback_parse_failed", "error", err)
		return false
	}
	if strings.TrimSpace(data.Signature) == "" || strings.TrimSpace(data.OutOrderID) == "" || strings.TrimSpace(data.TokenOrderID) == "" {
		log.Debugw("tokenpay_callback_not_matched")
		return false
	}

	log.Infow("tokenpay_callback_received",
		"out_order_id", data.OutOrderID,
		"token_order_id", data.TokenOrderID,
		"status", data.Status,
		"raw_body", callbackRawBodyForLog(body),
	)

	payment, err := h.PaymentRepo.GetByGatewayOrderNo(data.OutOrderID)
	if err != nil || payment == nil {
		payment, err = h.PaymentRepo.GetLatestByProviderRef(data.TokenOrderID)
		if err != nil || payment == nil {
			log.Warnw("tokenpay_callback_payment_not_found", "out_order_id", data.OutOrderID, "token_order_id", data.TokenOrderID, "error", err)
			c.String(200, constants.TokenPayCallbackFail)
			return true
		}
	}

	channel, err := h.PaymentChannelRepo.GetByID(payment.ChannelID)
	if err != nil || channel == nil {
		log.Warnw("tokenpay_callback_channel_not_found", "payment_id", payment.ID, "channel_id", payment.ChannelID, "error", err)
		c.String(200, constants.TokenPayCallbackFail)
		return true
	}
	if strings.ToLower(strings.TrimSpace(channel.ProviderType)) != constants.PaymentProviderTokenpay {
		log.Warnw("tokenpay_callback_provider_invalid", "provider_type", channel.ProviderType)
		c.String(200, constants.TokenPayCallbackFail)
		return true
	}

	cfg, err := tokenpay.ParseConfig(channel.ConfigJSON)
	if err != nil {
		log.Warnw("tokenpay_callback_config_parse_failed", "error", err)
		c.String(200, constants.TokenPayCallbackFail)
		return true
	}
	if strings.TrimSpace(cfg.Currency) == "" {
		cfg.Currency = tokenpay.DefaultCurrency
	}
	if err := tokenpay.ValidateConfig(cfg); err != nil {
		log.Warnw("tokenpay_callback_config_invalid", "error", err)
		c.String(200, constants.TokenPayCallbackFail)
		return true
	}
	if err := tokenpay.VerifyCallback(data, cfg.NotifySecret); err != nil {
		log.Warnw("tokenpay_callback_signature_invalid", "error", err)
		c.String(200, constants.TokenPayCallbackFail)
		return true
	}

	amount := models.Money{}
	if parsed := tokenpay.ParseAmount(data.ActualAmount); parsed != "" {
		if parsedAmount, parseErr := decimal.NewFromString(parsed); parseErr == nil {
			amount = models.NewMoneyFromDecimal(parsedAmount)
		}
	}
	callbackInput := service.PaymentCallbackInput{
		PaymentID:   payment.ID,
		OrderNo:     strings.TrimSpace(data.OutOrderID),
		ChannelID:   channel.ID,
		Status:      tokenpay.ToPaymentStatus(data.Status),
		ProviderRef: strings.TrimSpace(data.TokenOrderID),
		Amount:      amount,
		Currency:    strings.TrimSpace(data.BaseCurrency),
		PaidAt:      tokenpay.ParsePaidAt(data.PayTime),
		Payload:     models.JSON(data.Raw),
	}

	updated, err := h.PaymentService.HandleCallback(callbackInput)
	if err != nil {
		log.Warnw("tokenpay_callback_handle_failed", "payment_id", payment.ID, "error", err)
		c.String(200, constants.TokenPayCallbackFail)
		return true
	}

	log.Infow("tokenpay_callback_processed",
		"payment_id", payment.ID,
		"order_no", callbackInput.OrderNo,
		"provider_ref", callbackInput.ProviderRef,
		"status", updated.Status,
	)
	c.String(200, constants.TokenPayCallbackSuccess)
	return true
}
