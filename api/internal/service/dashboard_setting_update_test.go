package service

import (
	"testing"

	"github.com/dujiao-next/internal/constants"
)

func TestUpdateDashboardSettingNormalized(t *testing.T) {
	repo := newMockSettingRepo()
	svc := NewSettingService(repo)

	input := map[string]interface{}{
		"alert": map[string]interface{}{
			"low_stock_threshold":              9999,
			"out_of_stock_products_threshold":  -2,
			"pending_payment_orders_threshold": "200001",
			"payments_failed_threshold":        0,
		},
		"ranking": map[string]interface{}{
			"top_products_limit": 999,
			"top_channels_limit": -1,
		},
	}

	result, err := svc.Update(constants.SettingKeyDashboardConfig, input)
	if err != nil {
		t.Fatalf("update dashboard config failed: %v", err)
	}

	alert, ok := result["alert"].(map[string]interface{})
	if !ok {
		t.Fatalf("invalid alert payload type: %T", result["alert"])
	}
	ranking, ok := result["ranking"].(map[string]interface{})
	if !ok {
		t.Fatalf("invalid ranking payload type: %T", result["ranking"])
	}

	assertSettingIntValue(t, alert, "low_stock_threshold", 5)
	assertSettingIntValue(t, alert, "out_of_stock_products_threshold", 1)
	assertSettingIntValue(t, alert, "pending_payment_orders_threshold", 20)
	assertSettingIntValue(t, alert, "payments_failed_threshold", 10)
	assertSettingIntValue(t, ranking, "top_products_limit", 5)
	assertSettingIntValue(t, ranking, "top_channels_limit", 5)
}

func TestUpdateDashboardSettingFallbackWhenMissing(t *testing.T) {
	repo := newMockSettingRepo()
	svc := NewSettingService(repo)

	result, err := svc.Update(constants.SettingKeyDashboardConfig, map[string]interface{}{})
	if err != nil {
		t.Fatalf("update dashboard config failed: %v", err)
	}

	alert, ok := result["alert"].(map[string]interface{})
	if !ok {
		t.Fatalf("invalid alert payload type: %T", result["alert"])
	}
	ranking, ok := result["ranking"].(map[string]interface{})
	if !ok {
		t.Fatalf("invalid ranking payload type: %T", result["ranking"])
	}

	assertSettingIntValue(t, alert, "low_stock_threshold", 5)
	assertSettingIntValue(t, alert, "out_of_stock_products_threshold", 1)
	assertSettingIntValue(t, alert, "pending_payment_orders_threshold", 20)
	assertSettingIntValue(t, alert, "payments_failed_threshold", 10)
	assertSettingIntValue(t, ranking, "top_products_limit", 5)
	assertSettingIntValue(t, ranking, "top_channels_limit", 5)
}

func assertSettingIntValue(t *testing.T, data map[string]interface{}, key string, expected int) {
	t.Helper()
	value, exists := data[key]
	if !exists {
		t.Fatalf("missing key %s", key)
	}
	parsed, err := parseSettingInt(value)
	if err != nil {
		t.Fatalf("parse key %s failed: %v", key, err)
	}
	if parsed != expected {
		t.Fatalf("unexpected value for %s, expected %d got %d", key, expected, parsed)
	}
}
