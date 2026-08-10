package service

import (
	"testing"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/payment/stripe"

	"github.com/shopspring/decimal"
)

func TestValidateChannelStripeOfficial(t *testing.T) {
	svc := &PaymentService{}
	channel := &models.PaymentChannel{
		ProviderType:    constants.PaymentProviderOfficial,
		ChannelType:     constants.PaymentChannelTypeStripe,
		InteractionMode: constants.PaymentInteractionRedirect,
		FeeRate:         models.NewMoneyFromDecimal(decimal.Zero),
		ConfigJSON: models.JSON{
			"secret_key":             "sk_test_123456",
			"webhook_secret":         "whsec_123456",
			"success_url":            "https://example.com/payment?stripe_return=1",
			"cancel_url":             "https://example.com/payment?stripe_cancel=1",
			"api_base_url":           "https://api.stripe.com",
			"payment_method_types":   []string{"card"},
			"publishable_key":        "pk_test_123456",
			"webhook_tolerance_secs": 300,
		},
	}
	if err := svc.ValidateChannel(channel); err != nil {
		t.Fatalf("validate stripe channel failed: %v", err)
	}
}

func TestValidateChannelStripeInvalidInteractionMode(t *testing.T) {
	svc := &PaymentService{}
	channel := &models.PaymentChannel{
		ProviderType:    constants.PaymentProviderOfficial,
		ChannelType:     constants.PaymentChannelTypeStripe,
		InteractionMode: constants.PaymentInteractionQR,
		FeeRate:         models.NewMoneyFromDecimal(decimal.Zero),
		ConfigJSON: models.JSON{
			"secret_key":           "sk_test_123456",
			"webhook_secret":       "whsec_123456",
			"success_url":          "https://example.com/payment?stripe_return=1",
			"cancel_url":           "https://example.com/payment?stripe_cancel=1",
			"api_base_url":         "https://api.stripe.com",
			"payment_method_types": []string{"card"},
		},
	}
	if err := svc.ValidateChannel(channel); err == nil {
		t.Fatalf("expected invalid interaction mode error")
	}
}

func TestMapStripeGatewayError(t *testing.T) {
	if got := mapStripeGatewayError(stripe.ErrConfigInvalid); got != ErrPaymentChannelConfigInvalid {
		t.Fatalf("expected config invalid mapping, got: %v", got)
	}
	if got := mapStripeGatewayError(stripe.ErrRequestFailed); got != ErrPaymentGatewayRequestFailed {
		t.Fatalf("expected request failed mapping, got: %v", got)
	}
	if got := mapStripeGatewayError(stripe.ErrSignatureInvalid); got != ErrPaymentGatewayResponseInvalid {
		t.Fatalf("expected signature invalid mapping, got: %v", got)
	}
	if got := mapStripeGatewayError(stripe.ErrResponseInvalid); got != ErrPaymentGatewayResponseInvalid {
		t.Fatalf("expected response invalid mapping, got: %v", got)
	}
}
