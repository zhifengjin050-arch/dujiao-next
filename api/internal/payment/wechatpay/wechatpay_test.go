package wechatpay

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/dujiao-next/internal/constants"
)

func TestParseAndValidateConfigRedirect(t *testing.T) {
	cfg, err := ParseConfig(map[string]interface{}{
		"appid":                "wx1234567890",
		"mchid":                "1900000109",
		"merchant_serial_no":   "ABC123456789",
		"merchant_private_key": buildTestPrivateKey(),
		"api_v3_key":           "12345678901234567890123456789012",
		"notify_url":           "https://example.com/api/v1/payments/callback",
		"h5_redirect_url":      "https://example.com/pay",
		"h5_type":              "wap",
		"h5_wap_url":           "https://example.com",
		"h5_wap_name":          "Demo",
	})
	if err != nil {
		t.Fatalf("parse config failed: %v", err)
	}
	if cfg.BaseURL != defaultBaseURL {
		t.Fatalf("base url should fallback to default, got: %s", cfg.BaseURL)
	}
	if err := ValidateConfig(cfg, constants.PaymentInteractionRedirect); err != nil {
		t.Fatalf("validate config failed: %v", err)
	}
}

func TestValidateConfigRequireH5RedirectURL(t *testing.T) {
	cfg, err := ParseConfig(map[string]interface{}{
		"appid":                "wx1234567890",
		"mchid":                "1900000109",
		"merchant_serial_no":   "ABC123456789",
		"merchant_private_key": buildTestPrivateKey(),
		"api_v3_key":           "12345678901234567890123456789012",
		"notify_url":           "https://example.com/api/v1/payments/callback",
	})
	if err != nil {
		t.Fatalf("parse config failed: %v", err)
	}
	if err := ValidateConfig(cfg, constants.PaymentInteractionRedirect); err == nil {
		t.Fatalf("expected h5_redirect_url required error")
	}
}

func TestValidateConfigSupportsQRWithoutH5RedirectURL(t *testing.T) {
	cfg, err := ParseConfig(map[string]interface{}{
		"appid":                "wx1234567890",
		"mchid":                "1900000109",
		"merchant_serial_no":   "ABC123456789",
		"merchant_private_key": buildTestPrivateKey(),
		"api_v3_key":           "12345678901234567890123456789012",
		"notify_url":           "https://example.com/api/v1/payments/callback",
	})
	if err != nil {
		t.Fatalf("parse config failed: %v", err)
	}
	if err := ValidateConfig(cfg, constants.PaymentInteractionQR); err != nil {
		t.Fatalf("unexpected validate error: %v", err)
	}
}

func TestValidateConfigInvalidAPIV3KeyLength(t *testing.T) {
	cfg, err := ParseConfig(map[string]interface{}{
		"appid":                "wx1234567890",
		"mchid":                "1900000109",
		"merchant_serial_no":   "ABC123456789",
		"merchant_private_key": buildTestPrivateKey(),
		"api_v3_key":           "short-key",
		"notify_url":           "https://example.com/api/v1/payments/callback",
	})
	if err != nil {
		t.Fatalf("parse config failed: %v", err)
	}
	if err := ValidateConfig(cfg, constants.PaymentInteractionQR); err == nil {
		t.Fatalf("expected invalid api_v3_key length error")
	}
}

func TestCreatePaymentH5Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/v3/pay/transactions/h5" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode body failed: %v", err)
		}
		if payload["out_trade_no"] != "ORDER-1001" {
			t.Fatalf("unexpected out_trade_no: %v", payload["out_trade_no"])
		}
		amount, ok := payload["amount"].(map[string]interface{})
		if !ok {
			t.Fatalf("amount payload missing")
		}
		if amount["total"] != float64(1050) {
			t.Fatalf("unexpected amount total: %v", amount["total"])
		}
		if amount["currency"] != "CNY" {
			t.Fatalf("unexpected amount currency: %v", amount["currency"])
		}
		sceneInfo, ok := payload["scene_info"].(map[string]interface{})
		if !ok {
			t.Fatalf("scene_info missing")
		}
		if sceneInfo["payer_client_ip"] != "127.0.0.1" {
			t.Fatalf("unexpected payer_client_ip: %v", sceneInfo["payer_client_ip"])
		}
		h5Info, ok := sceneInfo["h5_info"].(map[string]interface{})
		if !ok {
			t.Fatalf("h5_info missing")
		}
		if h5Info["type"] != "WAP" {
			t.Fatalf("unexpected h5 type: %v", h5Info["type"])
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"h5_url":"https://wx.tenpay.com/cgi-bin/mmpayweb-bin/checkmweb?prepay_id=wx123"}`))
	}))
	defer server.Close()

	cfg, err := ParseConfig(map[string]interface{}{
		"appid":                "wx1234567890",
		"mchid":                "1900000109",
		"merchant_serial_no":   "ABC123456789",
		"merchant_private_key": buildTestPrivateKey(),
		"api_v3_key":           "12345678901234567890123456789012",
		"notify_url":           "https://example.com/api/v1/payments/callback",
		"h5_redirect_url":      "https://example.com/pay/result",
		"h5_type":              "WAP",
		"h5_wap_url":           "https://m.example.com",
		"h5_wap_name":          "demo",
		"base_url":             server.URL,
	})
	if err != nil {
		t.Fatalf("parse config failed: %v", err)
	}

	result, err := CreatePayment(context.Background(), cfg, CreateInput{
		OrderNo:     "ORDER-1001",
		Amount:      "10.50",
		Currency:    "USD",
		Description: "????",
		ClientIP:    "127.0.0.1",
	}, constants.PaymentInteractionRedirect)
	if err != nil {
		t.Fatalf("create payment failed: %v", err)
	}
	if result.PayURL == "" {
		t.Fatalf("expected pay url")
	}
	if result.QRCode != "" {
		t.Fatalf("h5 payment should not contain qrcode")
	}
	parsedURL, err := url.Parse(result.PayURL)
	if err != nil {
		t.Fatalf("parse pay url failed: %v", err)
	}
	if parsedURL.Query().Get("prepay_id") != "wx123" {
		t.Fatalf("missing prepay_id in pay url: %s", result.PayURL)
	}
	if parsedURL.Query().Get("redirect_url") != "https://example.com/pay/result" {
		t.Fatalf("unexpected redirect_url: %s", parsedURL.Query().Get("redirect_url"))
	}
}

func TestCreatePaymentNativeSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3/pay/transactions/native" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"code_url":"weixin://wxpay/bizpayurl?pr=mocked"}`))
	}))
	defer server.Close()

	cfg, err := ParseConfig(map[string]interface{}{
		"appid":                "wx1234567890",
		"mchid":                "1900000109",
		"merchant_serial_no":   "ABC123456789",
		"merchant_private_key": buildTestPrivateKey(),
		"api_v3_key":           "12345678901234567890123456789012",
		"notify_url":           "https://example.com/api/v1/payments/callback",
		"base_url":             server.URL,
	})
	if err != nil {
		t.Fatalf("parse config failed: %v", err)
	}

	result, err := CreatePayment(context.Background(), cfg, CreateInput{
		OrderNo:     "ORDER-1002",
		Amount:      "1.00",
		Currency:    "CNY",
		Description: "????",
		ClientIP:    "127.0.0.1",
	}, constants.PaymentInteractionQR)
	if err != nil {
		t.Fatalf("create payment failed: %v", err)
	}
	if result.QRCode != "weixin://wxpay/bizpayurl?pr=mocked" {
		t.Fatalf("unexpected qrcode: %s", result.QRCode)
	}
	if result.PayURL != "" {
		t.Fatalf("native payment should not contain pay url")
	}
}

func TestCreatePaymentResponseInvalid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":"INVALID_REQUEST"}`))
	}))
	defer server.Close()

	cfg, err := ParseConfig(map[string]interface{}{
		"appid":                "wx1234567890",
		"mchid":                "1900000109",
		"merchant_serial_no":   "ABC123456789",
		"merchant_private_key": buildTestPrivateKey(),
		"api_v3_key":           "12345678901234567890123456789012",
		"notify_url":           "https://example.com/api/v1/payments/callback",
		"h5_redirect_url":      "https://example.com/pay/result",
		"base_url":             server.URL,
	})
	if err != nil {
		t.Fatalf("parse config failed: %v", err)
	}

	_, err = CreatePayment(context.Background(), cfg, CreateInput{
		OrderNo:     "ORDER-1003",
		Amount:      "2.00",
		Currency:    "CNY",
		Description: "????",
		ClientIP:    "127.0.0.1",
	}, constants.PaymentInteractionRedirect)
	if !errors.Is(err, ErrResponseInvalid) {
		t.Fatalf("expected ErrResponseInvalid, got: %v", err)
	}
}

func TestQueryOrderByOutTradeNoSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/v3/pay/transactions/out-trade-no/ORDER-2001" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("mchid") != "1900000109" {
			t.Fatalf("unexpected mchid: %s", r.URL.Query().Get("mchid"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"out_trade_no":"ORDER-2001",
			"transaction_id":"4200002001202602100000001",
			"trade_state":"SUCCESS",
			"success_time":"2026-02-10T10:00:00+08:00",
			"amount":{"total":1234,"currency":"CNY"},
			"attach":"1001"
		}`))
	}))
	defer server.Close()

	cfg, err := ParseConfig(map[string]interface{}{
		"appid":                "wx1234567890",
		"mchid":                "1900000109",
		"merchant_serial_no":   "ABC123456789",
		"merchant_private_key": buildTestPrivateKey(),
		"api_v3_key":           "12345678901234567890123456789012",
		"notify_url":           "https://example.com/api/v1/payments/callback",
		"base_url":             server.URL,
	})
	if err != nil {
		t.Fatalf("parse config failed: %v", err)
	}
	result, err := QueryOrderByOutTradeNo(context.Background(), cfg, "ORDER-2001")
	if err != nil {
		t.Fatalf("query order failed: %v", err)
	}
	if result.Status != constants.PaymentStatusSuccess {
		t.Fatalf("unexpected status: %s", result.Status)
	}
	if result.Amount != "12.34" {
		t.Fatalf("unexpected amount: %s", result.Amount)
	}
	if result.TransactionID == "" {
		t.Fatalf("expected transaction id")
	}
}

func TestToPaymentStatus(t *testing.T) {
	tests := []struct {
		name     string
		trade    string
		expect   string
		expectOK bool
	}{
		{name: "Success", trade: wechatTradeStateSuccess, expect: constants.PaymentStatusSuccess, expectOK: true},
		{name: "Refund", trade: wechatTradeStateRefund, expect: constants.PaymentStatusSuccess, expectOK: true},
		{name: "NotPay", trade: wechatTradeStateNotPay, expect: constants.PaymentStatusPending, expectOK: true},
		{name: "UserPaying", trade: wechatTradeStateUserPaying, expect: constants.PaymentStatusPending, expectOK: true},
		{name: "PayError", trade: wechatTradeStatePayError, expect: constants.PaymentStatusFailed, expectOK: true},
		{name: "LowercaseInput", trade: strings.ToLower(wechatTradeStateSuccess), expect: constants.PaymentStatusSuccess, expectOK: true},
		{name: "Unknown", trade: "UNKNOWN", expect: "", expectOK: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			status, ok := ToPaymentStatus(tc.trade)
			if ok != tc.expectOK {
				t.Fatalf("unexpected ok: got %v, want %v", ok, tc.expectOK)
			}
			if status != tc.expect {
				t.Fatalf("unexpected status: got %s, want %s", status, tc.expect)
			}
		})
	}
}

func buildTestPrivateKey() string {
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
