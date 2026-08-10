package service

import (
	"testing"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/payment/paypal"
)

func TestMapPaypalStatus(t *testing.T) {
	status, ok := mapPaypalStatus("COMPLETED")
	if !ok || status != constants.PaymentStatusSuccess {
		t.Fatalf("expected success, got %s %v", status, ok)
	}
	status, ok = mapPaypalStatus("declined")
	if !ok || status != constants.PaymentStatusFailed {
		t.Fatalf("expected failed, got %s %v", status, ok)
	}
	status, ok = mapPaypalStatus("pending")
	if !ok || status != constants.PaymentStatusPending {
		t.Fatalf("expected pending, got %s %v", status, ok)
	}
	status, ok = mapPaypalStatus("unknown")
	if ok || status != "" {
		t.Fatalf("expected unknown mapping, got %s %v", status, ok)
	}
}

func TestAppendURLQuery(t *testing.T) {
	result := appendURLQuery("https://example.com/payment", map[string]string{
		"order_id":   "100",
		"payment_id": "200",
		"pp_return":  "1",
	})
	if result == "" {
		t.Fatalf("appendURLQuery returned empty result")
	}
	if result != "https://example.com/payment?order_id=100&payment_id=200&pp_return=1" &&
		result != "https://example.com/payment?order_id=100&pp_return=1&payment_id=200" &&
		result != "https://example.com/payment?payment_id=200&order_id=100&pp_return=1" &&
		result != "https://example.com/payment?payment_id=200&pp_return=1&order_id=100" &&
		result != "https://example.com/payment?pp_return=1&order_id=100&payment_id=200" &&
		result != "https://example.com/payment?pp_return=1&payment_id=200&order_id=100" {
		t.Fatalf("unexpected query result: %s", result)
	}
}

func TestPickFirstNonEmpty(t *testing.T) {
	if got := pickFirstNonEmpty("", " ", "abc", "def"); got != "abc" {
		t.Fatalf("expected abc, got %s", got)
	}
	if got := pickFirstNonEmpty("", " "); got != "" {
		t.Fatalf("expected empty value, got %s", got)
	}
}

func TestShouldMarkFulfilling(t *testing.T) {
	if shouldMarkFulfilling(nil) {
		t.Fatalf("nil order should not be fulfilling")
	}
	order := &models.Order{Items: []models.OrderItem{{FulfillmentType: constants.FulfillmentTypeAuto}}}
	if shouldMarkFulfilling(order) {
		t.Fatalf("auto items should not require fulfilling")
	}
	order = &models.Order{Items: []models.OrderItem{{FulfillmentType: constants.FulfillmentTypeManual}}}
	if !shouldMarkFulfilling(order) {
		t.Fatalf("manual items should require fulfilling")
	}
}

func TestBuildPaypalCallbackAmountSuccess(t *testing.T) {
	event := &paypal.WebhookEvent{
		Resource: map[string]interface{}{
			"amount": map[string]interface{}{
				"value":         "12.34",
				"currency_code": "usd",
			},
		},
	}

	amount, currency, err := buildPaypalCallbackAmount(event, constants.PaymentStatusSuccess)
	if err != nil {
		t.Fatalf("buildPaypalCallbackAmount should succeed, got: %v", err)
	}
	if amount.String() != "12.34" {
		t.Fatalf("unexpected amount: %s", amount.String())
	}
	if currency != "USD" {
		t.Fatalf("unexpected currency: %s", currency)
	}
}

func TestBuildPaypalCallbackAmountSuccessMissingAmount(t *testing.T) {
	event := &paypal.WebhookEvent{
		Resource: map[string]interface{}{
			"status": "COMPLETED",
		},
	}

	_, _, err := buildPaypalCallbackAmount(event, constants.PaymentStatusSuccess)
	if err == nil {
		t.Fatalf("buildPaypalCallbackAmount should fail when success callback misses amount")
	}
}

func TestBuildPaypalCallbackAmountSuccessInvalidAmount(t *testing.T) {
	event := &paypal.WebhookEvent{
		Resource: map[string]interface{}{
			"amount": map[string]interface{}{
				"value":         "invalid",
				"currency_code": "USD",
			},
		},
	}

	_, _, err := buildPaypalCallbackAmount(event, constants.PaymentStatusSuccess)
	if err == nil {
		t.Fatalf("buildPaypalCallbackAmount should fail when amount is invalid")
	}
}

func TestBuildPaypalCallbackAmountPendingAllowEmpty(t *testing.T) {
	event := &paypal.WebhookEvent{
		Resource: map[string]interface{}{
			"status": "PENDING",
		},
	}

	amount, currency, err := buildPaypalCallbackAmount(event, constants.PaymentStatusPending)
	if err != nil {
		t.Fatalf("buildPaypalCallbackAmount should allow empty amount for pending status, got: %v", err)
	}
	if !amount.Decimal.IsZero() {
		t.Fatalf("expected zero amount for pending status, got: %s", amount.String())
	}
	if currency != "" {
		t.Fatalf("expected empty currency for pending status, got: %s", currency)
	}
}
