package epay

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dujiao-next/internal/constants"
)

func TestCreatePaymentV1HandlesDoubleEncodedJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Accept-Encoding"); got != "identity" {
			t.Fatalf("Accept-Encoding = %s, want identity", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("\"{\\\"code\\\":1,\\\"msg\\\":\\\"success\\\",\\\"trade_no\\\":\\\"T20260322001\\\",\\\"payurl\\\":\\\"https://pay.example.com/v1\\\"}\""))
	}))
	defer server.Close()

	cfg := &Config{
		GatewayURL:  server.URL,
		EpayVersion: VersionV1,
		MerchantID:  "1001",
		MerchantKey: "key-001",
		NotifyURL:   "https://api.example.com/api/v1/payments/callback",
		ReturnURL:   "https://shop.example.com/pay",
		SignType:    epaySignTypeMD5,
	}
	cfg.Normalize()

	result, err := CreatePayment(context.Background(), cfg, CreateInput{
		OrderNo:     "DJP-V1-001",
		Amount:      "10.00",
		Subject:     "????",
		ChannelType: constants.PaymentChannelTypeAlipay,
		ClientIP:    "127.0.0.1",
		NotifyURL:   cfg.NotifyURL,
		ReturnURL:   cfg.ReturnURL,
	})
	if err != nil {
		t.Fatalf("CreatePayment v1 failed: %v", err)
	}
	if result.TradeNo != "T20260322001" {
		t.Fatalf("trade no = %s", result.TradeNo)
	}
	if result.PayURL != "https://pay.example.com/v1" {
		t.Fatalf("pay url = %s", result.PayURL)
	}
	if result.Raw == nil || result.Raw["code"] != float64(1) {
		t.Fatalf("raw response should be decoded into object, got %#v", result.Raw)
	}
}

func TestCreatePaymentV2HandlesDoubleEncodedJSON(t *testing.T) {
	privateKeyPEM, publicKeyPEM := generateEpayRSAKeyPair(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Accept-Encoding"); got != "identity" {
			t.Fatalf("Accept-Encoding = %s, want identity", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("\"{\\\"code\\\":0,\\\"msg\\\":\\\"success\\\",\\\"trade_no\\\":\\\"T20260322002\\\",\\\"pay_type\\\":\\\"qrcode\\\",\\\"pay_info\\\":\\\"https://pay.example.com/v2-qr\\\"}\""))
	}))
	defer server.Close()

	cfg := &Config{
		GatewayURL:  server.URL,
		EpayVersion: VersionV2,
		MerchantID:  "1002",
		PrivateKey:  privateKeyPEM,
		PublicKey:   publicKeyPEM,
		NotifyURL:   "https://api.example.com/api/v1/payments/callback",
		ReturnURL:   "https://shop.example.com/pay",
		SignType:    epaySignTypeRSA,
	}
	cfg.Normalize()

	result, err := CreatePayment(context.Background(), cfg, CreateInput{
		OrderNo:     "DJP-V2-001",
		Amount:      "20.00",
		Subject:     "????2",
		ChannelType: constants.PaymentChannelTypeWechat,
		ClientIP:    "127.0.0.1",
		NotifyURL:   cfg.NotifyURL,
		ReturnURL:   cfg.ReturnURL,
	})
	if err != nil {
		t.Fatalf("CreatePayment v2 failed: %v", err)
	}
	if result.TradeNo != "T20260322002" {
		t.Fatalf("trade no = %s", result.TradeNo)
	}
	if result.QRCode != "https://pay.example.com/v2-qr" {
		t.Fatalf("qr code = %s", result.QRCode)
	}
	if result.PayType != constants.EpayPayTypeQRCode {
		t.Fatalf("pay type = %s", result.PayType)
	}
	if result.Raw == nil || result.Raw["code"] != float64(0) {
		t.Fatalf("raw response should be decoded into object, got %#v", result.Raw)
	}
}
