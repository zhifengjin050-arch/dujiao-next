package paypal

import (
	"testing"

	"github.com/dujiao-next/internal/constants"
)

func TestValidateConfig(t *testing.T) {
	cfg := &Config{
		ClientID:     "cid",
		ClientSecret: "secret",
		BaseURL:      "https://api-m.sandbox.paypal.com",
		ReturnURL:    "https://example.com/payment?order_id=1",
		CancelURL:    "https://example.com/payment?order_id=1",
		WebhookID:    "WH-123456",
	}
	if err := ValidateConfig(cfg); err != nil {
		t.Fatalf("ValidateConfig should pass, got: %v", err)
	}
}

func TestValidateConfigWebhookIDRequired(t *testing.T) {
	cfg := &Config{
		ClientID:     "cid",
		ClientSecret: "secret",
		BaseURL:      "https://api-m.sandbox.paypal.com",
		ReturnURL:    "https://example.com/payment?order_id=1",
		CancelURL:    "https://example.com/payment?order_id=1",
	}
	if err := ValidateConfig(cfg); err == nil {
		t.Fatalf("ValidateConfig should fail when webhook_id is empty")
	}
}

func TestParseConfigAndNormalize(t *testing.T) {
	raw := map[string]interface{}{
		"client_id":     " cid ",
		"client_secret": " secret ",
		"base_url":      "https://api-m.sandbox.paypal.com/",
		"return_url":    "https://example.com/return",
		"cancel_url":    "https://example.com/cancel",
	}
	cfg, err := ParseConfig(raw)
	if err != nil {
		t.Fatalf("ParseConfig error: %v", err)
	}
	if cfg.ClientID != "cid" {
		t.Fatalf("client id not normalized, got: %s", cfg.ClientID)
	}
	if cfg.BaseURL != "https://api-m.sandbox.paypal.com" {
		t.Fatalf("base url not normalized, got: %s", cfg.BaseURL)
	}
	if cfg.UserAction == "" {
		t.Fatalf("user action should have default value")
	}
}

func TestToPaymentStatus(t *testing.T) {
	tests := []struct {
		name           string
		eventType      string
		resourceStatus string
		expectStatus   string
		expectOK       bool
	}{
		{name: "EventCompleted", eventType: paypalEventCaptureCompleted, resourceStatus: "", expectStatus: constants.PaymentStatusSuccess, expectOK: true},
		{name: "EventPending", eventType: paypalEventCapturePending, resourceStatus: "", expectStatus: constants.PaymentStatusPending, expectOK: true},
		{name: "ResourceDeclined", eventType: "", resourceStatus: paypalResourceStatusDeclined, expectStatus: constants.PaymentStatusFailed, expectOK: true},
		{name: "ResourceCreated", eventType: "", resourceStatus: paypalResourceStatusCreated, expectStatus: constants.PaymentStatusPending, expectOK: true},
		{name: "Unknown", eventType: "UNKNOWN", resourceStatus: "UNKNOWN", expectStatus: "", expectOK: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status, ok := ToPaymentStatus(tc.eventType, tc.resourceStatus)
			if ok != tc.expectOK {
				t.Fatalf("unexpected ok: got %v, want %v", ok, tc.expectOK)
			}
			if status != tc.expectStatus {
				t.Fatalf("unexpected status: got %s, want %s", status, tc.expectStatus)
			}
		})
	}
}

func TestWebhookEventHelpers(t *testing.T) {
	event := &WebhookEvent{
		EventType: "PAYMENT.CAPTURE.COMPLETED",
		Resource: map[string]interface{}{
			"supplementary_data": map[string]interface{}{
				"related_ids": map[string]interface{}{
					"order_id": "ORDER-123",
				},
			},
			"amount": map[string]interface{}{
				"value":         "10.00",
				"currency_code": "USD",
			},
			"create_time": "2026-02-09T12:00:00Z",
			"status":      "COMPLETED",
		},
	}
	if got := event.RelatedOrderID(); got != "ORDER-123" {
		t.Fatalf("unexpected order id: %s", got)
	}
	value, currency := event.CaptureAmount()
	if value != "10.00" || currency != "USD" {
		t.Fatalf("unexpected amount info: %s %s", value, currency)
	}
	if event.PaidAt() == nil {
		t.Fatalf("PaidAt should parse time")
	}
	if status := event.ResourceStatus(); status != "COMPLETED" {
		t.Fatalf("unexpected resource status: %s", status)
	}
}

func TestWebhookEventHelpersCaptureAmountFallback(t *testing.T) {
	event := &WebhookEvent{
		EventType: "CHECKOUT.ORDER.COMPLETED",
		Resource: map[string]interface{}{
			"purchase_units": []interface{}{
				map[string]interface{}{
					"amount": map[string]interface{}{
						"value":         "88.66",
						"currency_code": "USD",
					},
				},
			},
		},
	}

	value, currency := event.CaptureAmount()
	if value != "88.66" || currency != "USD" {
		t.Fatalf("unexpected fallback amount info: %s %s", value, currency)
	}
}
