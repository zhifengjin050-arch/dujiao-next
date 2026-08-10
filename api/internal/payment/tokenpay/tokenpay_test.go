package tokenpay

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSignPayloadIgnoreSignatureAndEmpty(t *testing.T) {
	payload := map[string]interface{}{
		"OutOrderId": "ORDER-1001",
		"Status":     1,
		"Empty":      "",
		"Signature":  "will-be-ignored",
	}
	got := SignPayload(payload, "secret")
	if got != "f8a446bc7d18188839fcc25918ec2078" {
		t.Fatalf("sign mismatch, got=%s", got)
	}
}

func TestParseAndVerifyCallback(t *testing.T) {
	raw := map[string]interface{}{
		"OutOrderId":   "ORDER-2001",
		"Id":           "tp-abc-1",
		"Status":       1,
		"ActualAmount": "15.00",
	}
	raw["Signature"] = SignPayload(raw, "notify-secret")
	body, err := json.Marshal(raw)
	if err != nil {
		t.Fatalf("marshal callback body failed: %v", err)
	}

	callback, err := ParseCallback(body)
	if err != nil {
		t.Fatalf("parse callback failed: %v", err)
	}
	if callback.OutOrderID != "ORDER-2001" {
		t.Fatalf("out order id mismatch, got=%s", callback.OutOrderID)
	}
	if callback.TokenOrderID != "tp-abc-1" {
		t.Fatalf("token order id mismatch, got=%s", callback.TokenOrderID)
	}
	if callback.Status != 1 {
		t.Fatalf("status mismatch, got=%d", callback.Status)
	}
	if err := VerifyCallback(callback, "notify-secret"); err != nil {
		t.Fatalf("verify callback failed: %v", err)
	}
	if err := VerifyCallback(callback, "wrong-secret"); err == nil {
		t.Fatalf("verify callback should fail with wrong secret")
	}
}

func TestCreatePayment(t *testing.T) {
	var gotBody map[string]interface{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method want POST got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/CreateOrder") {
			t.Fatalf("path mismatch, got=%s", r.URL.Path)
		}
		decoder := json.NewDecoder(r.Body)
		decoder.UseNumber()
		if err := decoder.Decode(&gotBody); err != nil {
			t.Fatalf("decode request body failed: %v", err)
		}
		_, _ = w.Write([]byte(`{"success":true,"message":"ok","data":"https://pay.example.com/p/1","info":{"Id":"tp-1001","QrCodeBase64":"data:image/png;base64,abc","QrCodeLink":"https://pay.example.com/qr/1"}}`))
	}))
	defer srv.Close()

	cfg := &Config{
		GatewayURL:   srv.URL,
		NotifySecret: "notify-secret",
		Currency:     "TRX",
	}
	result, err := CreatePayment(context.Background(), cfg, CreateInput{
		OutOrderID:   "ORDER-3001",
		OrderUserKey: "10001",
		ActualAmount: "15.00",
		NotifyURL:    "https://api.example.com/api/v1/payments/callback",
		RedirectURL:  "https://shop.example.com/pay?order_no=ORDER-3001",
	})
	if err != nil {
		t.Fatalf("create payment failed: %v", err)
	}
	if result.PayURL != "https://pay.example.com/p/1" {
		t.Fatalf("pay url mismatch, got=%s", result.PayURL)
	}
	if result.TokenOrderID != "tp-1001" {
		t.Fatalf("token order id mismatch, got=%s", result.TokenOrderID)
	}
	if strings.TrimSpace(gotBody["Signature"].(string)) == "" {
		t.Fatalf("signature should not be empty")
	}
}
