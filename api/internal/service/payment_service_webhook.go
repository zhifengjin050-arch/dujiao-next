package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/payment/paypal"
	"github.com/dujiao-next/internal/payment/stripe"
	"github.com/dujiao-next/internal/payment/wechatpay"
	"github.com/dujiao-next/internal/repository"

	"github.com/shopspring/decimal"
)

// HandlePaypalWebhook ?? PayPal webhook?
func (s *PaymentService) HandlePaypalWebhook(input WebhookCallbackInput) (*models.Payment, string, error) {
	log := paymentLogger(
		"provider", constants.PaymentChannelTypePaypal,
		"channel_id", input.ChannelID,
		"body_size", len(input.Body),
	)
	if input.ChannelID == 0 {
		log.Warnw("payment_webhook_invalid_channel_id")
		return nil, "", ErrPaymentInvalid
	}
	channel, err := s.channelRepo.GetByID(input.ChannelID)
	if err != nil {
		log.Errorw("payment_webhook_channel_fetch_failed", "error", err)
		return nil, "", ErrPaymentUpdateFailed
	}
	if channel == nil {
		log.Warnw("payment_webhook_channel_not_found")
		return nil, "", ErrPaymentChannelNotFound
	}
	providerType := strings.ToLower(strings.TrimSpace(channel.ProviderType))
	channelType := strings.ToLower(strings.TrimSpace(channel.ChannelType))
	if providerType != constants.PaymentProviderOfficial || channelType != constants.PaymentChannelTypePaypal {
		log.Warnw("payment_webhook_provider_mismatch",
			"provider_type", channel.ProviderType,
			"channel_type", channel.ChannelType,
		)
		return nil, "", ErrPaymentProviderNotSupported
	}

	cfg, err := paypal.ParseConfig(channel.ConfigJSON)
	if err != nil {
		log.Warnw("payment_webhook_config_parse_failed", "error", err)
		return nil, "", fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
	}
	if err := paypal.ValidateConfig(cfg); err != nil {
		log.Warnw("payment_webhook_config_invalid", "error", err)
		return nil, "", fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
	}

	event, err := paypal.ParseWebhookEvent(input.Body)
	if err != nil {
		log.Warnw("payment_webhook_payload_invalid", "error", err)
		return nil, "", ErrPaymentGatewayResponseInvalid
	}
	log.Infow("payment_webhook_event_parsed",
		"event_type", event.EventType,
		"event_id", event.ID,
	)

	ctx, cancel := detachOutboundRequestContext(input.Context)
	defer cancel()
	headers := make(http.Header)
	for key, value := range input.Headers {
		headers.Set(key, value)
	}
	if err := paypal.VerifyWebhookSignature(ctx, cfg, headers, event.Raw); err != nil {
		log.Warnw("payment_webhook_signature_invalid",
			"event_type", event.EventType,
			"event_id", event.ID,
			"error", err,
		)
		switch {
		case errors.Is(err, paypal.ErrConfigInvalid):
			return nil, event.EventType, fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
		case errors.Is(err, paypal.ErrAuthFailed), errors.Is(err, paypal.ErrRequestFailed):
			return nil, event.EventType, ErrPaymentGatewayRequestFailed
		default:
			return nil, event.EventType, ErrPaymentGatewayResponseInvalid
		}
	}

	var payment *models.Payment
	invoiceID := strings.TrimSpace(event.RelatedInvoiceID())
	if invoiceID != "" {
		payment, err = s.paymentRepo.GetByGatewayOrderNo(invoiceID)
		if err != nil {
			log.Warnw("payment_webhook_gateway_order_lookup_failed", "invoice_id", invoiceID, "error", err)
		}
	}
	paypalOrderID := strings.TrimSpace(event.RelatedOrderID())
	if payment == nil && paypalOrderID != "" {
		payment, err = s.paymentRepo.GetLatestByProviderRef(paypalOrderID)
		if err != nil {
			log.Warnw("payment_webhook_provider_ref_lookup_failed", "provider_ref", paypalOrderID, "error", err)
		}
	}
	if payment == nil {
		log.Infow("payment_webhook_payment_not_found",
			"provider_ref", paypalOrderID,
			"event_type", event.EventType,
			"event_id", event.ID,
		)
		return nil, event.EventType, nil
	}

	status, ok := paypal.ToPaymentStatus(event.EventType, event.ResourceStatus())
	if !ok {
		log.Infow("payment_webhook_status_ignored",
			"payment_id", payment.ID,
			"provider_ref", paypalOrderID,
			"event_type", event.EventType,
			"event_id", event.ID,
		)
		return payment, event.EventType, nil
	}

	amount, amountCurrency, err := buildPaypalCallbackAmount(event, status)
	if err != nil {
		log.Warnw("payment_webhook_amount_invalid",
			"payment_id", payment.ID,
			"provider_ref", paypalOrderID,
			"event_type", event.EventType,
			"event_id", event.ID,
			"error", err,
		)
		return nil, event.EventType, err
	}

	resourceBytes, _ := json.Marshal(event.Raw)
	payloadMap := map[string]interface{}{}
	if len(resourceBytes) > 0 {
		_ = json.Unmarshal(resourceBytes, &payloadMap)
	}

	updated, err := s.HandleCallback(PaymentCallbackInput{
		PaymentID:   payment.ID,
		ChannelID:   channel.ID,
		Status:      status,
		ProviderRef: paypalOrderID,
		Amount:      amount,
		Currency:    strings.ToUpper(strings.TrimSpace(amountCurrency)),
		PaidAt:      event.PaidAt(),
		Payload:     models.JSON(payloadMap),
	})
	if err != nil {
		log.Errorw("payment_webhook_callback_apply_failed",
			"payment_id", payment.ID,
			"provider_ref", paypalOrderID,
			"event_type", event.EventType,
			"event_id", event.ID,
			"error", err,
		)
		return nil, event.EventType, err
	}
	log.Infow("payment_webhook_processed",
		"payment_id", updated.ID,
		"provider_ref", paypalOrderID,
		"event_type", event.EventType,
		"event_id", event.ID,
		"status", updated.Status,
	)
	return updated, event.EventType, nil
}

func buildPaypalCallbackAmount(event *paypal.WebhookEvent, status string) (models.Money, string, error) {
	amount := models.Money{}
	if event == nil {
		return amount, "", ErrPaymentGatewayResponseInvalid
	}

	amountValue, amountCurrency := event.CaptureAmount()
	amountValue = strings.TrimSpace(amountValue)
	amountCurrency = strings.ToUpper(strings.TrimSpace(amountCurrency))

	requiresAmount := strings.EqualFold(strings.TrimSpace(status), constants.PaymentStatusSuccess)
	if requiresAmount {
		if amountValue == "" || amountCurrency == "" {
			return amount, "", ErrPaymentGatewayResponseInvalid
		}
	}

	if amountValue == "" {
		if amountCurrency != "" {
			return amount, "", ErrPaymentGatewayResponseInvalid
		}
		return amount, "", nil
	}

	parsedAmount, err := decimal.NewFromString(amountValue)
	if err != nil || parsedAmount.Cmp(decimal.Zero) <= 0 {
		return amount, "", ErrPaymentGatewayResponseInvalid
	}
	if amountCurrency == "" {
		return amount, "", ErrPaymentGatewayResponseInvalid
	}

	amount = models.NewMoneyFromDecimal(parsedAmount)
	return amount, amountCurrency, nil
}

// HandleWechatWebhook ?????????
func (s *PaymentService) HandleWechatWebhook(input WebhookCallbackInput) (*models.Payment, string, error) {
	log := paymentLogger(
		"provider", constants.PaymentChannelTypeWechat,
		"channel_id", input.ChannelID,
		"body_size", len(input.Body),
	)
	ctx, cancel := detachOutboundRequestContext(input.Context)
	defer cancel()

	candidates, err := s.resolveWechatWebhookChannels(input.ChannelID)
	if err != nil {
		log.Warnw("payment_webhook_resolve_channels_failed", "error", err)
		return nil, "", err
	}

	var lastErr error
	for i := range candidates {
		channel := candidates[i]
		cfg, err := wechatpay.ParseConfig(channel.ConfigJSON)
		if err != nil {
			mappedErr := fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
			if input.ChannelID != 0 {
				return nil, "", mappedErr
			}
			lastErr = mappedErr
			continue
		}
		if err := wechatpay.ValidateConfig(cfg, channel.InteractionMode); err != nil {
			mappedErr := fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
			if input.ChannelID != 0 {
				return nil, "", mappedErr
			}
			lastErr = mappedErr
			continue
		}

		result, err := wechatpay.VerifyAndDecodeWebhook(ctx, cfg, input.Headers, input.Body)
		if err != nil {
			mappedErr := mapWechatGatewayError(err)
			if input.ChannelID != 0 {
				return nil, "", mappedErr
			}
			lastErr = mappedErr
			continue
		}

		payment, err := s.findWechatWebhookPayment(channel.ID, result)
		if err != nil {
			if errors.Is(err, ErrPaymentNotFound) {
				log.Infow("payment_webhook_payment_not_found",
					"channel_id", channel.ID,
					"event_type", result.EventType,
					"provider_ref", result.TransactionID,
					"order_no", result.OrderNo,
				)
				return nil, result.EventType, nil
			}
			log.Warnw("payment_webhook_payment_lookup_failed",
				"channel_id", channel.ID,
				"event_type", result.EventType,
				"provider_ref", result.TransactionID,
				"order_no", result.OrderNo,
				"error", err,
			)
			return nil, result.EventType, err
		}
		if payment == nil {
			log.Infow("payment_webhook_payment_not_found",
				"channel_id", channel.ID,
				"event_type", result.EventType,
				"provider_ref", result.TransactionID,
				"order_no", result.OrderNo,
			)
			return nil, result.EventType, nil
		}

		updated, err := s.handleWechatWebhookCallback(channel.ID, payment, result)
		if err != nil {
			log.Warnw("payment_webhook_callback_apply_failed",
				"channel_id", channel.ID,
				"payment_id", payment.ID,
				"event_type", result.EventType,
				"provider_ref", result.TransactionID,
				"order_no", result.OrderNo,
				"error", err,
			)
			return nil, result.EventType, err
		}
		log.Infow("payment_webhook_processed",
			"channel_id", channel.ID,
			"payment_id", updated.ID,
			"event_type", result.EventType,
			"provider_ref", result.TransactionID,
			"order_no", result.OrderNo,
			"status", updated.Status,
		)
		return updated, result.EventType, nil
	}

	if lastErr != nil {
		log.Warnw("payment_webhook_verify_failed_all_channels", "error", lastErr)
		return nil, "", lastErr
	}
	log.Warnw("payment_webhook_no_channel_matched")
	return nil, "", ErrPaymentGatewayResponseInvalid
}

// HandleStripeWebhook ?? Stripe webhook?
func (s *PaymentService) HandleStripeWebhook(input WebhookCallbackInput) (*models.Payment, string, error) {
	log := paymentLogger(
		"provider", constants.PaymentChannelTypeStripe,
		"channel_id", input.ChannelID,
		"body_size", len(input.Body),
	)
	candidates, err := s.resolveStripeWebhookChannels(input.ChannelID)
	if err != nil {
		log.Warnw("payment_webhook_resolve_channels_failed", "error", err)
		return nil, "", err
	}

	var lastErr error
	for i := range candidates {
		channel := candidates[i]
		cfg, err := stripe.ParseConfig(channel.ConfigJSON)
		if err != nil {
			mappedErr := fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
			if input.ChannelID != 0 {
				return nil, "", mappedErr
			}
			lastErr = mappedErr
			continue
		}
		if err := stripe.ValidateConfig(cfg); err != nil {
			mappedErr := fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
			if input.ChannelID != 0 {
				return nil, "", mappedErr
			}
			lastErr = mappedErr
			continue
		}

		result, err := stripe.VerifyAndParseWebhook(cfg, input.Headers, input.Body, time.Now())
		if err != nil {
			mappedErr := mapStripeGatewayError(err)
			if input.ChannelID != 0 {
				return nil, "", mappedErr
			}
			lastErr = mappedErr
			continue
		}
		log.Infow("payment_webhook_event_parsed",
			"channel_id", channel.ID,
			"event_type", result.EventType,
			"event_id", result.EventID,
			"provider_ref", result.ProviderRef,
			"order_no", result.OrderNo,
		)

		payment, err := s.findStripeWebhookPayment(channel.ID, result)
		if err != nil {
			if errors.Is(err, ErrPaymentNotFound) {
				log.Infow("payment_webhook_payment_not_found",
					"channel_id", channel.ID,
					"event_type", result.EventType,
					"event_id", result.EventID,
					"provider_ref", result.ProviderRef,
					"order_no", result.OrderNo,
				)
				return nil, result.EventType, nil
			}
			log.Warnw("payment_webhook_payment_lookup_failed",
				"channel_id", channel.ID,
				"event_type", result.EventType,
				"event_id", result.EventID,
				"provider_ref", result.ProviderRef,
				"order_no", result.OrderNo,
				"error", err,
			)
			return nil, result.EventType, err
		}
		if payment == nil {
			log.Infow("payment_webhook_payment_not_found",
				"channel_id", channel.ID,
				"event_type", result.EventType,
				"event_id", result.EventID,
				"provider_ref", result.ProviderRef,
				"order_no", result.OrderNo,
			)
			return nil, result.EventType, nil
		}

		updated, err := s.handleStripeWebhookCallback(channel.ID, payment, result)
		if err != nil {
			log.Warnw("payment_webhook_callback_apply_failed",
				"channel_id", channel.ID,
				"payment_id", payment.ID,
				"event_type", result.EventType,
				"event_id", result.EventID,
				"provider_ref", result.ProviderRef,
				"order_no", result.OrderNo,
				"error", err,
			)
			return nil, result.EventType, err
		}
		log.Infow("payment_webhook_processed",
			"channel_id", channel.ID,
			"payment_id", updated.ID,
			"event_type", result.EventType,
			"event_id", result.EventID,
			"provider_ref", result.ProviderRef,
			"order_no", result.OrderNo,
			"status", updated.Status,
		)
		return updated, result.EventType, nil
	}

	if lastErr != nil {
		log.Warnw("payment_webhook_verify_failed_all_channels", "error", lastErr)
		return nil, "", lastErr
	}
	log.Warnw("payment_webhook_no_channel_matched")
	return nil, "", ErrPaymentGatewayResponseInvalid
}

func (s *PaymentService) resolveStripeWebhookChannels(channelID uint) ([]models.PaymentChannel, error) {
	if channelID != 0 {
		channel, err := s.channelRepo.GetByID(channelID)
		if err != nil {
			return nil, ErrPaymentUpdateFailed
		}
		if channel == nil {
			return nil, ErrPaymentChannelNotFound
		}
		providerType := strings.ToLower(strings.TrimSpace(channel.ProviderType))
		channelType := strings.ToLower(strings.TrimSpace(channel.ChannelType))
		if providerType != constants.PaymentProviderOfficial || channelType != constants.PaymentChannelTypeStripe {
			return nil, ErrPaymentProviderNotSupported
		}
		return []models.PaymentChannel{*channel}, nil
	}

	channels, _, err := s.channelRepo.List(repository.PaymentChannelListFilter{
		ProviderType: constants.PaymentProviderOfficial,
		ChannelType:  constants.PaymentChannelTypeStripe,
		ActiveOnly:   true,
	})
	if err != nil {
		return nil, ErrPaymentUpdateFailed
	}
	if len(channels) == 0 {
		return nil, ErrPaymentChannelNotFound
	}
	return channels, nil
}

func (s *PaymentService) findStripeWebhookPayment(channelID uint, result *stripe.WebhookResult) (*models.Payment, error) {
	if result == nil {
		return nil, ErrPaymentInvalid
	}
	if orderNo := strings.TrimSpace(result.OrderNo); orderNo != "" {
		payment, err := s.paymentRepo.GetByGatewayOrderNo(orderNo)
		if err != nil {
			return nil, ErrPaymentUpdateFailed
		}
		if payment != nil && payment.ChannelID == channelID {
			return payment, nil
		}
	}
	for _, ref := range []string{
		strings.TrimSpace(result.ProviderRef),
		strings.TrimSpace(result.SessionID),
		strings.TrimSpace(result.PaymentIntentID),
	} {
		if ref == "" {
			continue
		}
		payment, err := s.paymentRepo.GetLatestByProviderRef(ref)
		if err != nil {
			return nil, ErrPaymentUpdateFailed
		}
		if payment == nil {
			continue
		}
		if payment.ChannelID != channelID {
			continue
		}
		return payment, nil
	}
	return nil, ErrPaymentNotFound
}

func (s *PaymentService) handleStripeWebhookCallback(channelID uint, payment *models.Payment, result *stripe.WebhookResult) (*models.Payment, error) {
	if payment == nil || result == nil {
		return nil, ErrPaymentInvalid
	}
	amount := models.Money{}
	if strings.TrimSpace(result.Amount) != "" {
		parsed, err := decimal.NewFromString(strings.TrimSpace(result.Amount))
		if err == nil {
			amount = models.NewMoneyFromDecimal(parsed)
		}
	}
	payload := models.JSON{}
	if result.Raw != nil {
		payload = models.JSON(result.Raw)
	}
	status := strings.TrimSpace(result.Status)
	if status == "" {
		status = constants.PaymentStatusPending
	}
	callbackInput := PaymentCallbackInput{
		PaymentID: payment.ID,
		ChannelID: channelID,
		Status:    status,
		ProviderRef: pickFirstNonEmpty(
			strings.TrimSpace(result.ProviderRef),
			strings.TrimSpace(result.SessionID),
			strings.TrimSpace(result.PaymentIntentID),
			strings.TrimSpace(payment.ProviderRef),
		),
		Amount:   amount,
		Currency: strings.ToUpper(strings.TrimSpace(result.Currency)),
		PaidAt:   result.PaidAt,
		Payload:  payload,
	}
	return s.HandleCallback(callbackInput)
}

func (s *PaymentService) resolveWechatWebhookChannels(channelID uint) ([]models.PaymentChannel, error) {
	if channelID != 0 {
		channel, err := s.channelRepo.GetByID(channelID)
		if err != nil {
			return nil, ErrPaymentUpdateFailed
		}
		if channel == nil {
			return nil, ErrPaymentChannelNotFound
		}
		providerType := strings.ToLower(strings.TrimSpace(channel.ProviderType))
		channelType := strings.ToLower(strings.TrimSpace(channel.ChannelType))
		if providerType != constants.PaymentProviderOfficial || channelType != constants.PaymentChannelTypeWechat {
			return nil, ErrPaymentProviderNotSupported
		}
		return []models.PaymentChannel{*channel}, nil
	}

	channels, _, err := s.channelRepo.List(repository.PaymentChannelListFilter{
		ProviderType: constants.PaymentProviderOfficial,
		ChannelType:  constants.PaymentChannelTypeWechat,
		ActiveOnly:   true,
	})
	if err != nil {
		return nil, ErrPaymentUpdateFailed
	}
	if len(channels) == 0 {
		return nil, ErrPaymentChannelNotFound
	}
	return channels, nil
}

func (s *PaymentService) findWechatWebhookPayment(channelID uint, result *wechatpay.WebhookResult) (*models.Payment, error) {
	if result == nil {
		return nil, ErrPaymentInvalid
	}
	if orderNo := strings.TrimSpace(result.OrderNo); orderNo != "" {
		payment, err := s.paymentRepo.GetByGatewayOrderNo(orderNo)
		if err != nil {
			return nil, ErrPaymentUpdateFailed
		}
		if payment != nil && payment.ChannelID == channelID {
			return payment, nil
		}
	}
	if txID := strings.TrimSpace(result.TransactionID); txID != "" {
		payment, err := s.paymentRepo.GetLatestByProviderRef(txID)
		if err != nil {
			return nil, ErrPaymentUpdateFailed
		}
		if payment != nil && payment.ChannelID == channelID {
			return payment, nil
		}
	}
	return nil, ErrPaymentNotFound
}

func (s *PaymentService) handleWechatWebhookCallback(channelID uint, payment *models.Payment, result *wechatpay.WebhookResult) (*models.Payment, error) {
	if payment == nil || result == nil {
		return nil, ErrPaymentInvalid
	}
	amount := models.Money{}
	if strings.TrimSpace(result.Amount) != "" {
		parsed, err := decimal.NewFromString(strings.TrimSpace(result.Amount))
		if err == nil {
			amount = models.NewMoneyFromDecimal(parsed)
		}
	}
	payload := models.JSON{}
	if result.Raw != nil {
		payload = models.JSON(result.Raw)
	}
	callbackInput := PaymentCallbackInput{
		PaymentID:   payment.ID,
		ChannelID:   channelID,
		Status:      strings.TrimSpace(result.Status),
		ProviderRef: pickFirstNonEmpty(strings.TrimSpace(result.TransactionID), strings.TrimSpace(payment.ProviderRef)),
		Amount:      amount,
		Currency:    strings.ToUpper(strings.TrimSpace(result.Currency)),
		PaidAt:      result.PaidAt,
		Payload:     payload,
	}
	return s.HandleCallback(callbackInput)
}

func mapWechatGatewayError(err error) error {
	switch {
	case errors.Is(err, wechatpay.ErrConfigInvalid):
		return ErrPaymentChannelConfigInvalid
	case errors.Is(err, wechatpay.ErrRequestFailed):
		return ErrPaymentGatewayRequestFailed
	case errors.Is(err, wechatpay.ErrSignatureInvalid), errors.Is(err, wechatpay.ErrResponseInvalid):
		return ErrPaymentGatewayResponseInvalid
	default:
		return ErrPaymentGatewayRequestFailed
	}
}

func mapStripeGatewayError(err error) error {
	switch {
	case errors.Is(err, stripe.ErrConfigInvalid):
		return ErrPaymentChannelConfigInvalid
	case errors.Is(err, stripe.ErrRequestFailed):
		return ErrPaymentGatewayRequestFailed
	case errors.Is(err, stripe.ErrSignatureInvalid), errors.Is(err, stripe.ErrResponseInvalid):
		return ErrPaymentGatewayResponseInvalid
	default:
		return ErrPaymentGatewayRequestFailed
	}
}
