package service

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"testing"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/payment/wechatpay"

	"github.com/shopspring/decimal"
)

func TestValidateChannelWechatOfficial(t *testing.T) {
	svc := &PaymentService{}
	channel := &models.PaymentChannel{
		ProviderType:    constants.PaymentProviderOfficial,
		ChannelType:     constants.PaymentChannelTypeWechat,
		InteractionMode: constants.PaymentInteractionRedirect,
		FeeRate:         models.NewMoneyFromDecimal(decimal.Zero),
		ConfigJSON: models.JSON{
			"appid":                "wx1234567890",
			"mchid":                "1900000109",
			"merchant_serial_no":   "ABC123456789",
			"merchant_private_key": buildWechatTestPrivateKey(),
			"api_v3_key":           "12345678901234567890123456789012",
			"notify_url":           "https://example.com/api/v1/payments/callback",
			"h5_redirect_url":      "https://example.com/pay",
		},
	}
	if err := svc.ValidateChannel(channel); err != nil {
		t.Fatalf("validate wechat channel failed: %v", err)
	}
}

func TestValidateChannelWechatInvalidInteractionMode(t *testing.T) {
	svc := &PaymentService{}
	channel := &models.PaymentChannel{
		ProviderType:    constants.PaymentProviderOfficial,
		ChannelType:     constants.PaymentChannelTypeWechat,
		InteractionMode: constants.PaymentInteractionWAP,
		FeeRate:         models.NewMoneyFromDecimal(decimal.Zero),
		ConfigJSON: models.JSON{
			"appid":                "wx1234567890",
			"mchid":                "1900000109",
			"merchant_serial_no":   "ABC123456789",
			"merchant_private_key": buildWechatTestPrivateKey(),
			"api_v3_key":           "12345678901234567890123456789012",
			"notify_url":           "https://example.com/api/v1/payments/callback",
			"h5_redirect_url":      "https://example.com/pay",
		},
	}
	if err := svc.ValidateChannel(channel); err == nil {
		t.Fatalf("expected invalid interaction mode error")
	}
}

func buildWechatTestPrivateKey() string {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	privateKeyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		panic(err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateKeyDER}))
}

func TestMapWechatGatewayError(t *testing.T) {
	if got := mapWechatGatewayError(wechatpay.ErrConfigInvalid); got != ErrPaymentChannelConfigInvalid {
		t.Fatalf("expected config invalid mapping, got: %v", got)
	}
	if got := mapWechatGatewayError(wechatpay.ErrRequestFailed); got != ErrPaymentGatewayRequestFailed {
		t.Fatalf("expected request failed mapping, got: %v", got)
	}
	if got := mapWechatGatewayError(wechatpay.ErrSignatureInvalid); got != ErrPaymentGatewayResponseInvalid {
		t.Fatalf("expected signature invalid mapping, got: %v", got)
	}
	if got := mapWechatGatewayError(wechatpay.ErrResponseInvalid); got != ErrPaymentGatewayResponseInvalid {
		t.Fatalf("expected response invalid mapping, got: %v", got)
	}
}

func TestShouldUseCNYPaymentCurrency(t *testing.T) {
	if shouldUseCNYPaymentCurrency(nil) {
		t.Fatalf("nil channel should not force CNY")
	}
	if !shouldUseCNYPaymentCurrency(&models.PaymentChannel{ProviderType: constants.PaymentProviderOfficial, ChannelType: constants.PaymentChannelTypeWechat}) {
		t.Fatalf("official wechat should force CNY")
	}
	if !shouldUseCNYPaymentCurrency(&models.PaymentChannel{ProviderType: constants.PaymentProviderOfficial, ChannelType: constants.PaymentChannelTypeAlipay}) {
		t.Fatalf("official alipay should force CNY")
	}
	if shouldUseCNYPaymentCurrency(&models.PaymentChannel{ProviderType: constants.PaymentProviderOfficial, ChannelType: constants.PaymentChannelTypePaypal}) {
		t.Fatalf("official paypal should not force CNY")
	}
}
