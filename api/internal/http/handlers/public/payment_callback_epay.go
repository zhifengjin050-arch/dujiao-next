package public

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/http/handlers/shared"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/payment/epay"
	"github.com/dujiao-next/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

func (h *Handler) HandleEpayCallback(c *gin.Context) bool {
	log := shared.RequestLog(c)
	form, err := parseCallbackForm(c)
	if err != nil {
		log.Warnw("epay_callback_form_parse_failed", "error", err)
		return false
	}
	outTradeNo := strings.TrimSpace(getFirstValue(form, "out_trade_no"))
	pid := strings.TrimSpace(getFirstValue(form, "pid"))
	if pid == "" || outTradeNo == "" {
		log.Debugw("epay_callback_not_matched", "reason", "missing_pid_or_out_trade_no")
		return false
	}
	if strings.TrimSpace(getFirstValue(form, "trade_status")) == "" {
		log.Debugw("epay_callback_not_matched", "reason", "missing_trade_status")
		return false
	}
	log.Infow("epay_callback_received",
		"client_ip", c.ClientIP(),
		"out_trade_no", outTradeNo,
		"trade_no", strings.TrimSpace(getFirstValue(form, "trade_no")),
		"trade_status", strings.TrimSpace(getFirstValue(form, "trade_status")),
		"raw_form", callbackRawFormForLog(form),
	)
	payment, err := h.PaymentRepo.GetByGatewayOrderNo(outTradeNo)
	if err != nil || payment == nil {
		log.Warnw("epay_callback_payment_not_found", "out_trade_no", outTradeNo, "error", err)
		c.String(200, constants.EpayCallbackFail)
		return true
	}
	channel, err := h.PaymentChannelRepo.GetByID(payment.ChannelID)
	if err != nil || channel == nil {
		log.Warnw("epay_callback_channel_not_found",
			"payment_id", payment.ID,
			"channel_id", payment.ChannelID,
			"error", err,
		)
		c.String(200, constants.EpayCallbackFail)
		return true
	}
	if strings.ToLower(strings.TrimSpace(channel.ProviderType)) != constants.PaymentProviderEpay {
		log.Warnw("epay_callback_provider_invalid",
			"payment_id", payment.ID,
			"channel_id", channel.ID,
			"provider_type", channel.ProviderType,
		)
		c.String(200, constants.EpayCallbackFail)
		return true
	}
	cfg, err := epay.ParseConfig(channel.ConfigJSON)
	if err != nil {
		log.Warnw("epay_callback_config_parse_failed",
			"payment_id", payment.ID,
			"channel_id", channel.ID,
			"error", err,
		)
		c.String(200, constants.EpayCallbackFail)
		return true
	}
	if err := epay.ValidateConfig(cfg); err != nil {
		log.Warnw("epay_callback_config_invalid",
			"payment_id", payment.ID,
			"channel_id", channel.ID,
			"error", err,
		)
		c.String(200, constants.EpayCallbackFail)
		return true
	}
	if err := epay.VerifyCallback(cfg, form); err != nil {
		log.Warnw("epay_callback_signature_invalid",
			"payment_id", payment.ID,
			"channel_id", channel.ID,
			"error", err,
		)
		h.enqueuePaymentExceptionAlert(c, models.JSON{
			"alert_type":  "epay_signature_invalid",
			"alert_level": "error",
			"payment_id":  fmt.Sprintf("%d", payment.ID),
			"message":     strings.TrimSpace(err.Error()),
			"provider":    constants.PaymentProviderEpay,
		})
		c.String(200, constants.EpayCallbackFail)
		return true
	}
	if err := epay.VerifyCallbackOwnership(cfg, form); err != nil {
		log.Warnw("epay_callback_ownership_invalid",
			"payment_id", payment.ID,
			"channel_id", channel.ID,
			"error", err,
		)
		h.enqueuePaymentExceptionAlert(c, models.JSON{
			"alert_type":  "epay_ownership_invalid",
			"alert_level": "error",
			"payment_id":  fmt.Sprintf("%d", payment.ID),
			"message":     strings.TrimSpace(err.Error()),
			"provider":    constants.PaymentProviderEpay,
		})
		c.String(200, constants.EpayCallbackFail)
		return true
	}
	input, err := parseEpayCallback(form, payment.ID)
	if err != nil {
		log.Warnw("epay_callback_parse_failed",
			"payment_id", payment.ID,
			"channel_id", channel.ID,
			"error", err,
		)
		h.enqueuePaymentExceptionAlert(c, models.JSON{
			"alert_type":  "epay_callback_parse_failed",
			"alert_level": "error",
			"payment_id":  fmt.Sprintf("%d", payment.ID),
			"message":     strings.TrimSpace(err.Error()),
			"provider":    constants.PaymentProviderEpay,
		})
		c.String(200, constants.EpayCallbackFail)
		return true
	}
	input.ChannelID = channel.ID
	updated, err := h.PaymentService.HandleCallback(*input)
	if err != nil {
		log.Warnw("epay_callback_handle_failed",
			"payment_id", payment.ID,
			"channel_id", channel.ID,
			"order_no", input.OrderNo,
			"provider_ref", input.ProviderRef,
			"status", input.Status,
			"error", err,
		)
		h.enqueuePaymentExceptionAlert(c, models.JSON{
			"alert_type":  "epay_callback_handle_failed",
			"alert_level": "error",
			"payment_id":  fmt.Sprintf("%d", payment.ID),
			"order_no":    strings.TrimSpace(input.OrderNo),
			"message":     strings.TrimSpace(err.Error()),
			"provider":    constants.PaymentProviderEpay,
		})
		c.String(200, constants.EpayCallbackFail)
		return true
	}
	log.Infow("epay_callback_processed",
		"payment_id", payment.ID,
		"channel_id", channel.ID,
		"order_no", input.OrderNo,
		"provider_ref", input.ProviderRef,
		"status", updated.Status,
	)
	c.String(200, constants.EpayCallbackSuccess)
	return true
}

func parseEpayCallback(form map[string][]string, paymentID uint) (*service.PaymentCallbackInput, error) {
	orderNo := strings.TrimSpace(getFirstValue(form, "out_trade_no"))
	tradeStatus := strings.TrimSpace(getFirstValue(form, "trade_status"))
	status := constants.PaymentStatusFailed
	if tradeStatus == constants.EpayTradeStatusSuccess {
		status = constants.PaymentStatusSuccess
	}
	amount := models.Money{}
	if money := strings.TrimSpace(getFirstValue(form, "money")); money != "" {
		parsed, err := decimal.NewFromString(money)
		if err != nil {
			return nil, service.ErrPaymentInvalid
		}
		amount = models.NewMoneyFromDecimal(parsed)
	}
	paidAt := parseEpayPaidAt(getFirstValue(form, "endtime"), getFirstValue(form, "addtime"))
	providerRef := strings.TrimSpace(getFirstValue(form, "trade_no"))
	if providerRef == "" {
		providerRef = strings.TrimSpace(getFirstValue(form, "api_trade_no"))
	}
	payload := make(map[string]interface{}, len(form))
	for key, values := range form {
		if len(values) > 0 {
			payload[key] = values[0]
		}
	}
	return &service.PaymentCallbackInput{
		PaymentID:   paymentID,
		OrderNo:     orderNo,
		Status:      status,
		ProviderRef: providerRef,
		Amount:      amount,
		PaidAt:      paidAt,
		Payload:     models.JSON(payload),
	}, nil
}

func parseEpayPaidAt(endTime, addTime string) *time.Time {
	for _, val := range []string{endTime, addTime} {
		val = strings.TrimSpace(val)
		if val == "" {
			continue
		}
		parsed, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			continue
		}
		t := time.Unix(parsed, 0)
		return &t
	}
	return nil
}
