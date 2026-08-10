package public

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/http/handlers/shared"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/payment/okpay"
	"github.com/dujiao-next/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

func (h *Handler) HandleOkpayCallback(c *gin.Context) bool {
	log := shared.RequestLog(c)

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return false
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

	data, err := okpay.ParseCallback(body)
	if err != nil {
		log.Debugw("okpay_callback_parse_failed", "error", err)
		return false
	}
	if data.Sign == "" || data.OrderID == "" || data.UniqueID == "" {
		log.Debugw("okpay_callback_not_matched")
		return false
	}

	log.Infow("okpay_callback_received",
		"order_id", data.OrderID,
		"unique_id", data.UniqueID,
		"type", data.Type,
		"status", data.PaymentStatus,
		"raw_body", callbackRawBodyForLog(body),
	)

	payment, err := h.PaymentRepo.GetByGatewayOrderNo(data.UniqueID)
	if err != nil || payment == nil {
		payment, err = h.PaymentRepo.GetLatestByProviderRef(data.OrderID)
		if err != nil || payment == nil {
			log.Warnw("okpay_callback_payment_not_found", "unique_id", data.UniqueID, "order_id", data.OrderID, "error", err)
			c.Data(200, "application/json", []byte(constants.OkpayCallbackFail))
			return true
		}
	}

	channel, err := h.PaymentChannelRepo.GetByID(payment.ChannelID)
	if err != nil || channel == nil {
		log.Warnw("okpay_callback_channel_not_found", "payment_id", payment.ID, "channel_id", payment.ChannelID, "error", err)
		c.Data(200, "application/json", []byte(constants.OkpayCallbackFail))
		return true
	}
	if strings.ToLower(strings.TrimSpace(channel.ProviderType)) != constants.PaymentProviderOkpay {
		log.Warnw("okpay_callback_provider_invalid", "provider_type", channel.ProviderType)
		c.Data(200, "application/json", []byte(constants.OkpayCallbackFail))
		return true
	}

	cfg, err := okpay.ParseConfig(channel.ConfigJSON)
	if err != nil {
		log.Warnw("okpay_callback_config_parse_failed", "error", err)
		c.Data(200, "application/json", []byte(constants.OkpayCallbackFail))
		return true
	}
	if strings.TrimSpace(cfg.Coin) == "" {
		cfg.Coin = okpay.ResolveCoin(channel.ChannelType)
	}
	if err := okpay.ValidateConfig(cfg); err != nil {
		log.Warnw("okpay_callback_config_invalid", "error", err)
		c.Data(200, "application/json", []byte(constants.OkpayCallbackFail))
		return true
	}
	if err := okpay.VerifyCallback(cfg, data); err != nil {
		log.Warnw("okpay_callback_signature_invalid", "error", err)
		h.enqueuePaymentExceptionAlert(c, models.JSON{
			"alert_type":  "okpay_signature_invalid",
			"alert_level": "error",
			"payment_id":  fmt.Sprintf("%d", payment.ID),
			"message":     strings.TrimSpace(err.Error()),
			"provider":    constants.PaymentProviderOkpay,
		})
		c.Data(200, "application/json", []byte(constants.OkpayCallbackFail))
		return true
	}
	if err := verifyOkpayCallbackAmount(payment, cfg, data); err != nil {
		log.Warnw("okpay_callback_amount_invalid", "error", err)
		h.enqueuePaymentExceptionAlert(c, models.JSON{
			"alert_type":  "okpay_callback_amount_invalid",
			"alert_level": "error",
			"payment_id":  fmt.Sprintf("%d", payment.ID),
			"message":     strings.TrimSpace(err.Error()),
			"provider":    constants.PaymentProviderOkpay,
		})
		c.Data(200, "application/json", []byte(constants.OkpayCallbackFail))
		return true
	}

	callbackInput := service.PaymentCallbackInput{
		PaymentID:   payment.ID,
		OrderNo:     strings.TrimSpace(data.UniqueID),
		ChannelID:   channel.ID,
		Status:      okpay.ToPaymentStatus(data.RequestStatus, data.PaymentStatus),
		ProviderRef: strings.TrimSpace(data.OrderID),
		Amount:      models.Money{},
		PaidAt:      ptrCallbackPaidAt(okpay.ToPaymentStatus(data.RequestStatus, data.PaymentStatus)),
		Payload:     buildOkpayPayload(data),
	}

	updated, err := h.PaymentService.HandleCallback(callbackInput)
	if err != nil {
		log.Warnw("okpay_callback_handle_failed", "payment_id", payment.ID, "error", err)
		h.enqueuePaymentExceptionAlert(c, models.JSON{
			"alert_type":  "okpay_callback_handle_failed",
			"alert_level": "error",
			"payment_id":  fmt.Sprintf("%d", payment.ID),
			"order_no":    strings.TrimSpace(data.UniqueID),
			"message":     strings.TrimSpace(err.Error()),
			"provider":    constants.PaymentProviderOkpay,
		})
		c.Data(200, "application/json", []byte(constants.OkpayCallbackFail))
		return true
	}

	log.Infow("okpay_callback_processed",
		"payment_id", payment.ID,
		"order_no", callbackInput.OrderNo,
		"provider_ref", callbackInput.ProviderRef,
		"status", updated.Status,
	)
	c.Data(200, "application/json", []byte(constants.OkpayCallbackSuccess))
	return true
}

func buildOkpayPayload(data *okpay.CallbackData) models.JSON {
	payload := models.JSON{}
	if data == nil {
		return payload
	}
	for key, value := range data.Raw {
		payload[key] = value
	}
	return payload
}

func ptrCallbackPaidAt(status string) *time.Time {
	if status != constants.PaymentStatusSuccess {
		return nil
	}
	now := time.Now()
	return &now
}

func verifyOkpayCallbackAmount(payment *models.Payment, cfg *okpay.Config, data *okpay.CallbackData) error {
	if payment == nil || cfg == nil || data == nil {
		return nil
	}
	callbackAmountRaw := strings.TrimSpace(data.Amount)
	if callbackAmountRaw == "" {
		return nil
	}
	expectedAmount, err := okpay.ConvertAmountByRate(payment.Amount.String(), cfg.ExchangeRate)
	if err != nil {
		return err
	}
	callbackAmount, err := decimal.NewFromString(callbackAmountRaw)
	if err != nil {
		return fmt.Errorf("invalid callback amount: %w", err)
	}
	if callbackAmount.Cmp(expectedAmount) != 0 {
		return fmt.Errorf("callback amount mismatch: got %s want %s", callbackAmount.StringFixed(8), expectedAmount.StringFixed(8))
	}
	return nil
}
