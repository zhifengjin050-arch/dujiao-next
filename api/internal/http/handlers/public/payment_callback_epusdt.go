package public

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/http/handlers/shared"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/payment/epusdt"
	"github.com/dujiao-next/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

// HandleEpusdtCallback ?? BEpusdt ??
func (h *Handler) HandleEpusdtCallback(c *gin.Context) bool {
	log := shared.RequestLog(c)

	// ?????
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return false
	}
	// ??????????
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	// ????? epusdt ????
	data, err := epusdt.ParseCallback(body)
	if err != nil {
		log.Debugw("epusdt_callback_parse_failed", "error", err)
		return false
	}

	// ????? trade_id ? order_id(epusdt ????)
	if data.TradeID == "" || data.OrderID == "" {
		log.Debugw("epusdt_callback_missing_fields", "trade_id", data.TradeID, "order_id", data.OrderID)
		return false
	}

	log.Infow("epusdt_callback_received",
		"trade_id", data.TradeID,
		"order_id", data.OrderID,
		"status", data.Status,
		"raw_body", callbackRawBodyForLog(body),
	)

	// ?? order_id(???????)??????,??? trade_id(??????)
	payment, err := h.PaymentRepo.GetByGatewayOrderNo(data.OrderID)
	if err != nil || payment == nil {
		payment, err = h.PaymentRepo.GetLatestByProviderRef(data.TradeID)
		if err != nil || payment == nil {
			log.Warnw("epusdt_callback_payment_not_found", "order_id", data.OrderID, "trade_id", data.TradeID, "error", err)
			c.String(200, constants.EpusdtCallbackFail)
			return true
		}
	}

	log.Debugw("epusdt_callback_payment_found", "payment_id", payment.ID, "channel_id", payment.ChannelID)

	// ??????
	channel, err := h.PaymentChannelRepo.GetByID(payment.ChannelID)
	if err != nil || channel == nil {
		log.Warnw("epusdt_callback_channel_not_found", "channel_id", payment.ChannelID, "error", err)
		c.String(200, constants.EpusdtCallbackFail)
		return true
	}

	// ????? epusdt ??
	if strings.ToLower(strings.TrimSpace(channel.ProviderType)) != constants.PaymentProviderEpusdt {
		log.Warnw("epusdt_callback_invalid_provider", "provider_type", channel.ProviderType)
		c.String(200, constants.EpusdtCallbackFail)
		return true
	}

	// ????
	cfg, err := epusdt.ParseConfig(channel.ConfigJSON)
	if err != nil {
		log.Warnw("epusdt_callback_config_parse_failed", "error", err)
		c.String(200, constants.EpusdtCallbackFail)
		return true
	}

	// ????
	if err := epusdt.VerifyCallback(cfg, data); err != nil {
		log.Warnw("epusdt_callback_signature_invalid", "error", err)
		c.String(200, constants.EpusdtCallbackFail)
		return true
	}

	log.Debugw("epusdt_callback_signature_verified")

	// ????
	status := epusdt.ToPaymentStatus(data.Status)

	// ??????
	amount := models.Money{}
	amountFloat := data.GetAmount()
	if amountFloat > 0 {
		amount = models.NewMoneyFromDecimal(decimal.NewFromFloat(amountFloat))
	}

	now := time.Now()

	// ???????????? JSON ??
	payloadBytes, _ := json.Marshal(data)
	var payload models.JSON
	_ = json.Unmarshal(payloadBytes, &payload)

	input := service.PaymentCallbackInput{
		PaymentID:   payment.ID,
		OrderNo:     data.OrderID,
		ChannelID:   channel.ID,
		Status:      status,
		ProviderRef: data.TradeID,
		Amount:      amount,
		PaidAt:      &now,
		Payload:     models.JSON(payload),
	}

	// ????
	if _, err := h.PaymentService.HandleCallback(input); err != nil {
		log.Errorw("epusdt_callback_handle_failed", "error", err)
		c.String(200, constants.EpusdtCallbackFail)
		return true
	}

	log.Infow("epusdt_callback_processed", "payment_id", payment.ID, "status", status)
	c.String(200, constants.EpusdtCallbackSuccess)
	return true
}
