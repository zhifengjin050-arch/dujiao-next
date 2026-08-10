package service

import (
	"errors"
	"fmt"
	"strings"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/payment/paypal"
	"github.com/dujiao-next/internal/payment/stripe"
	"github.com/dujiao-next/internal/payment/wechatpay"

	"github.com/shopspring/decimal"
)

func (s *PaymentService) CapturePayment(input CapturePaymentInput) (*models.Payment, error) {
	if input.PaymentID == 0 {
		return nil, ErrPaymentInvalid
	}
	payment, err := s.paymentRepo.GetByID(input.PaymentID)
	if err != nil {
		return nil, ErrPaymentUpdateFailed
	}
	if payment == nil {
		return nil, ErrPaymentNotFound
	}
	if payment.Status == constants.PaymentStatusSuccess {
		return payment, nil
	}

	channel, err := s.channelRepo.GetByID(payment.ChannelID)
	if err != nil {
		return nil, ErrPaymentUpdateFailed
	}
	if channel == nil {
		return nil, ErrPaymentChannelNotFound
	}

	providerType := strings.ToLower(strings.TrimSpace(channel.ProviderType))
	if providerType != constants.PaymentProviderOfficial {
		return nil, ErrPaymentProviderNotSupported
	}
	if strings.TrimSpace(payment.ProviderRef) == "" {
		return nil, ErrPaymentInvalid
	}

	channelType := strings.ToLower(strings.TrimSpace(channel.ChannelType))
	switch channelType {
	case constants.PaymentChannelTypePaypal:
		return s.capturePaypalPayment(input, payment, channel)
	case constants.PaymentChannelTypeStripe:
		return s.captureStripePayment(input, payment, channel)
	case constants.PaymentChannelTypeWechat:
		return s.captureWechatPayment(input, payment, channel)
	default:
		return nil, ErrPaymentProviderNotSupported
	}
}

func (s *PaymentService) capturePaypalPayment(input CapturePaymentInput, payment *models.Payment, channel *models.PaymentChannel) (*models.Payment, error) {
	cfg, err := paypal.ParseConfig(channel.ConfigJSON)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
	}
	if err := paypal.ValidateConfig(cfg); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
	}

	ctx, cancel := detachOutboundRequestContext(input.Context)
	defer cancel()

	captureResult, err := paypal.CaptureOrder(ctx, cfg, payment.ProviderRef)
	if err != nil {
		switch {
		case errors.Is(err, paypal.ErrConfigInvalid):
			return nil, fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
		case errors.Is(err, paypal.ErrAuthFailed), errors.Is(err, paypal.ErrRequestFailed):
			return nil, ErrPaymentGatewayRequestFailed
		case errors.Is(err, paypal.ErrResponseInvalid):
			return nil, ErrPaymentGatewayResponseInvalid
		default:
			return nil, ErrPaymentGatewayRequestFailed
		}
	}

	status, ok := mapPaypalStatus(strings.TrimSpace(captureResult.Status))
	if !ok {
		status = constants.PaymentStatusPending
	}
	payload := models.JSON{}
	if captureResult.Raw != nil {
		payload = models.JSON(captureResult.Raw)
	}
	amount := models.Money{}
	if strings.TrimSpace(captureResult.Amount) != "" {
		parsed, parseErr := decimal.NewFromString(strings.TrimSpace(captureResult.Amount))
		if parseErr == nil {
			amount = models.NewMoneyFromDecimal(parsed)
		}
	}
	callbackInput := PaymentCallbackInput{
		PaymentID:   payment.ID,
		OrderNo:     "",
		ChannelID:   channel.ID,
		Status:      status,
		ProviderRef: pickFirstNonEmpty(strings.TrimSpace(captureResult.OrderID), strings.TrimSpace(payment.ProviderRef)),
		Amount:      amount,
		Currency:    strings.ToUpper(strings.TrimSpace(captureResult.Currency)),
		PaidAt:      captureResult.PaidAt,
		Payload:     payload,
	}
	return s.HandleCallback(callbackInput)
}

func (s *PaymentService) captureWechatPayment(input CapturePaymentInput, payment *models.Payment, channel *models.PaymentChannel) (*models.Payment, error) {
	cfg, err := wechatpay.ParseConfig(channel.ConfigJSON)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
	}
	if err := wechatpay.ValidateConfig(cfg, channel.InteractionMode); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
	}

	ctx, cancel := detachOutboundRequestContext(input.Context)
	defer cancel()

	queryResult, err := wechatpay.QueryOrderByOutTradeNo(ctx, cfg, payment.ProviderRef)
	if err != nil {
		switch {
		case errors.Is(err, wechatpay.ErrConfigInvalid):
			return nil, fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
		case errors.Is(err, wechatpay.ErrRequestFailed):
			return nil, ErrPaymentGatewayRequestFailed
		case errors.Is(err, wechatpay.ErrResponseInvalid):
			return nil, ErrPaymentGatewayResponseInvalid
		default:
			return nil, ErrPaymentGatewayRequestFailed
		}
	}

	amount := models.Money{}
	if strings.TrimSpace(queryResult.Amount) != "" {
		parsed, parseErr := decimal.NewFromString(strings.TrimSpace(queryResult.Amount))
		if parseErr == nil {
			amount = models.NewMoneyFromDecimal(parsed)
		}
	}
	payload := models.JSON{}
	if queryResult.Raw != nil {
		payload = models.JSON(queryResult.Raw)
	}
	status := strings.TrimSpace(queryResult.Status)
	if status == "" {
		status = constants.PaymentStatusPending
	}
	callbackInput := PaymentCallbackInput{
		PaymentID:   payment.ID,
		ChannelID:   channel.ID,
		Status:      status,
		ProviderRef: pickFirstNonEmpty(strings.TrimSpace(queryResult.TransactionID), strings.TrimSpace(payment.ProviderRef)),
		Amount:      amount,
		Currency:    strings.ToUpper(strings.TrimSpace(queryResult.Currency)),
		PaidAt:      queryResult.PaidAt,
		Payload:     payload,
	}
	return s.HandleCallback(callbackInput)
}

func (s *PaymentService) captureStripePayment(input CapturePaymentInput, payment *models.Payment, channel *models.PaymentChannel) (*models.Payment, error) {
	cfg, err := stripe.ParseConfig(channel.ConfigJSON)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
	}
	if err := stripe.ValidateConfig(cfg); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPaymentChannelConfigInvalid, err)
	}

	ctx, cancel := detachOutboundRequestContext(input.Context)
	defer cancel()

	queryResult, err := stripe.QueryPayment(ctx, cfg, payment.ProviderRef)
	if err != nil {
		return nil, mapStripeGatewayError(err)
	}

	amount := models.Money{}
	if strings.TrimSpace(queryResult.Amount) != "" {
		parsed, parseErr := decimal.NewFromString(strings.TrimSpace(queryResult.Amount))
		if parseErr == nil {
			amount = models.NewMoneyFromDecimal(parsed)
		}
	}
	payload := models.JSON{}
	if queryResult.Raw != nil {
		payload = models.JSON(queryResult.Raw)
	}
	status := strings.TrimSpace(queryResult.Status)
	if status == "" {
		status = constants.PaymentStatusPending
	}
	callbackInput := PaymentCallbackInput{
		PaymentID: payment.ID,
		ChannelID: channel.ID,
		Status:    status,
		ProviderRef: pickFirstNonEmpty(
			strings.TrimSpace(queryResult.SessionID),
			strings.TrimSpace(queryResult.PaymentIntentID),
			strings.TrimSpace(payment.ProviderRef),
		),
		Amount:   amount,
		Currency: strings.ToUpper(strings.TrimSpace(queryResult.Currency)),
		PaidAt:   queryResult.PaidAt,
		Payload:  payload,
	}
	return s.HandleCallback(callbackInput)
}
