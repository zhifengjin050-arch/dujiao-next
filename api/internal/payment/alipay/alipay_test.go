package alipay

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/dujiao-next/internal/constants"
)

func TestParseAndValidateConfig(t *testing.T) {
	cfg, err := ParseConfig(map[string]interface{}{
		"app_id":              "2026000000000000",
		"private_key":         "-----BEGIN PRIVATE KEY-----abc",
		"alipay_public_key":   "-----BEGIN PUBLIC KEY-----abc",
		"gateway_url":         "https://openapi.alipay.com/gateway.do",
		"notify_url":          "https://example.com/api/v1/payments/callback",
		"return_url":          "https://example.com/pay/success",
		"sign_type":           "rsa2",
		"app_cert_sn":         "abc",
		"alipay_root_cert_sn": "root",
	})
	if err != nil {
		t.Fatalf("parse config failed: %v", err)
	}
	if err := ValidateConfig(cfg, constants.PaymentInteractionPage); err != nil {
		t.Fatalf("validate config failed: %v", err)
	}
	if cfg.SignType != "RSA2" {
		t.Fatalf("expected sign_type RSA2, got %s", cfg.SignType)
	}
}

func TestValidateConfigRequireReturnURL(t *testing.T) {
	cfg, err := ParseConfig(map[string]interface{}{
		"app_id":            "2026000000000000",
		"private_key":       "k",
		"alipay_public_key": "p",
		"gateway_url":       "https://openapi.alipay.com/gateway.do",
		"notify_url":        "https://example.com/api/v1/payments/callback",
	})
	if err != nil {
		t.Fatalf("parse config failed: %v", err)
	}
	if err := ValidateConfig(cfg, constants.PaymentInteractionWAP); err == nil {
		t.Fatalf("expected error for missing return_url")
	}
}

func TestValidateConfigSupportsQRWithoutReturnURL(t *testing.T) {
	cfg, err := ParseConfig(map[string]interface{}{
		"app_id":            "2026000000000000",
		"private_key":       "k",
		"alipay_public_key": "p",
		"gateway_url":       "https://openapi.alipay.com/gateway.do",
		"notify_url":        "https://example.com/api/v1/payments/callback",
	})
	if err != nil {
		t.Fatalf("parse config failed: %v", err)
	}
	if err := ValidateConfig(cfg, constants.PaymentInteractionQR); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreatePaymentPrecreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected post request, got %s", r.Method)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form failed: %v", err)
		}
		if r.Form.Get("method") != "alipay.trade.precreate" {
			t.Fatalf("expected precreate method, got %s", r.Form.Get("method"))
		}
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"alipay_trade_precreate_response": map[string]interface{}{
				"code":         "10000",
				"msg":          "Success",
				"out_trade_no": "ORDER-1",
				"trade_no":     "20260209000001",
				"qr_code":      "https://example.com/qr/abc",
			},
			"sign": "test-sign",
		})
	}))
	defer server.Close()

	cfg := buildTestConfig(server.URL)
	result, err := CreatePayment(context.Background(), cfg, CreateInput{
		OrderNo:   "ORDER-1",
		Amount:    "19.90",
		Subject:   "????",
		NotifyURL: cfg.NotifyURL,
	}, constants.PaymentInteractionQR)
	if err != nil {
		t.Fatalf("create payment failed: %v", err)
	}
	if result.QRCode == "" {
		t.Fatalf("expected qr code")
	}
	if result.OutTradeNo != "ORDER-1" {
		t.Fatalf("unexpected out_trade_no: %s", result.OutTradeNo)
	}
}

func TestCreatePaymentWAPReturnsPayURL(t *testing.T) {
	cfg := buildTestConfig("https://openapi.alipay.com/gateway.do")
	cfg.ReturnURL = "https://example.com/pay/return"
	result, err := CreatePayment(context.Background(), cfg, CreateInput{
		OrderNo:   "ORDER-2",
		Amount:    "99.99",
		Subject:   "????2",
		NotifyURL: cfg.NotifyURL,
		ReturnURL: cfg.ReturnURL,
	}, constants.PaymentInteractionWAP)
	if err != nil {
		t.Fatalf("create payment failed: %v", err)
	}
	if strings.TrimSpace(result.PayURL) == "" {
		t.Fatalf("expected pay url")
	}
	parsedURL, err := url.Parse(result.PayURL)
	if err != nil {
		t.Fatalf("parse pay url failed: %v", err)
	}
	if parsedURL.Query().Get("method") != "alipay.trade.wap.pay" {
		t.Fatalf("unexpected method: %s", parsedURL.Query().Get("method"))
	}
	if parsedURL.Query().Get("sign") == "" {
		t.Fatalf("expected sign in pay url")
	}
}

func TestCreatePaymentPrecreateResponseError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"alipay_trade_precreate_response": map[string]interface{}{
				"code":    "40004",
				"msg":     "Business Failed",
				"sub_msg": "ACQ.TRADE_NOT_EXIST",
			},
		})
	}))
	defer server.Close()

	cfg := buildTestConfig(server.URL)
	_, err := CreatePayment(context.Background(), cfg, CreateInput{
		OrderNo:   "ORDER-3",
		Amount:    "10.00",
		NotifyURL: cfg.NotifyURL,
	}, constants.PaymentInteractionQR)
	if err == nil {
		t.Fatalf("expected create payment error")
	}
	if !strings.Contains(err.Error(), ErrResponseInvalid.Error()) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestVerifyCallbackSuccess(t *testing.T) {
	cfg := buildTestConfig("https://openapi.alipay.com/gateway.do")
	form := map[string][]string{
		"notify_id":    {"notify-1"},
		"notify_type":  {"trade_status_sync"},
		"out_trade_no": []string{"ORDER-VERIFY-1"},
		"trade_no":     {"20260209000088"},
		"trade_status": []string{"TRADE_SUCCESS"},
		"total_amount": []string{"88.00"},
		"sign_type":    {"RSA2"},
	}
	content := buildSignContentFromForm(form)
	sign, err := signContent(content, cfg.PrivateKey, cfg.SignType)
	if err != nil {
		t.Fatalf("sign callback content failed: %v", err)
	}
	form["sign"] = []string{sign}
	if err := VerifyCallback(cfg, form); err != nil {
		t.Fatalf("verify callback failed: %v", err)
	}
}

func TestVerifyCallbackInvalidSign(t *testing.T) {
	cfg := buildTestConfig("https://openapi.alipay.com/gateway.do")
	form := map[string][]string{
		"notify_id":    {"notify-2"},
		"notify_type":  {"trade_status_sync"},
		"out_trade_no": []string{"ORDER-VERIFY-2"},
		"trade_no":     {"20260209000089"},
		"trade_status": []string{"TRADE_SUCCESS"},
		"total_amount": []string{"8.80"},
		"sign_type":    {"RSA2"},
		"sign":         {"invalid-sign"},
	}
	if err := VerifyCallback(cfg, form); err == nil {
		t.Fatalf("expected verify callback error")
	}
}

func TestVerifyCallbackOwnershipSuccess(t *testing.T) {
	cfg := buildTestConfig("https://openapi.alipay.com/gateway.do")
	form := map[string][]string{
		"app_id": []string{cfg.AppID},
	}
	if err := VerifyCallbackOwnership(cfg, form); err != nil {
		t.Fatalf("expected ownership verify success, got: %v", err)
	}
}

func TestVerifyCallbackOwnershipMissingAppID(t *testing.T) {
	cfg := buildTestConfig("https://openapi.alipay.com/gateway.do")
	form := map[string][]string{
		"notify_id": {"notify-3"},
	}
	if err := VerifyCallbackOwnership(cfg, form); err == nil {
		t.Fatalf("expected ownership verify error for missing app_id")
	}
}

func TestVerifyCallbackOwnershipAppIDMismatch(t *testing.T) {
	cfg := buildTestConfig("https://openapi.alipay.com/gateway.do")
	form := map[string][]string{
		"app_id": {"2026999999999999"},
	}
	if err := VerifyCallbackOwnership(cfg, form); err == nil {
		t.Fatalf("expected ownership verify error for app_id mismatch")
	}
}

func buildTestConfig(gatewayURL string) *Config {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic(err)
	}
	privateKeyDER, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		panic(err)
	}
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privateKeyDER})
	publicKeyDER, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		panic(err)
	}
	publicKeyPEM := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: publicKeyDER})
	return &Config{
		AppID:           "2026000000000000",
		PrivateKey:      string(privateKeyPEM),
		AlipayPublicKey: string(publicKeyPEM),
		GatewayURL:      gatewayURL,
		NotifyURL:       "https://example.com/api/v1/payments/callback",
		ReturnURL:       "https://example.com/pay/return",
		SignType:        "RSA2",
	}
}
