package okpay

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSignPayloadMatchesDocumentExample(t *testing.T) {
	payload := SignPayload(map[string]string{
		"amount":       "10",
		"callback_url": "http://127.0.0.1/callback",
		"coin":         "USDT",
		"name":         "test",
		"return_url":   "http://127.0.0.1",
		"unique_id":    "123456",
	}, "1", "123456")

	if payload["sign"] != "7465C8F4ED1BA0C8C2DB88E792374A65" {
		t.Fatalf("unexpected sign: %s", payload["sign"])
	}
}

func TestVerifyCallbackMatchesDocumentExample(t *testing.T) {
	cfg := &Config{
		MerchantID:    "1",
		MerchantToken: "123456",
	}
	body := "code=200&data[order_id]=ac7b86615fdb137576ae35879f7ed844&data[unique_id]=BWIN-20250922152023LDVNSyxLQko&data[pay_user_id]=7238234930&data[amount]=6.00000000&data[coin]=USDT&data[status]=1&data[type]=deposit&id=1&status=success&sign=95BE540FB7D1996770E2B4CDBC6F184D"
	data, err := ParseCallback([]byte(body))
	if err != nil {
		t.Fatalf("ParseCallback failed: %v", err)
	}
	if err := VerifyCallback(cfg, data); err != nil {
		t.Fatalf("VerifyCallback failed: %v", err)
	}
	if status := ToPaymentStatus(data.RequestStatus, data.PaymentStatus); status != "success" {
		t.Fatalf("unexpected payment status: %s", status)
	}
}

func TestCreatePayment(t *testing.T) {
	var receivedBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm failed: %v", err)
		}
		receivedBody = r.PostForm.Encode()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","code":200,"data":{"order_id":"OK-ORDER-1","pay_url":"https://pay.example.com/ok"}}`))
	}))
	defer server.Close()

	cfg := &Config{
		GatewayURL:    server.URL,
		MerchantID:    "shop-1",
		MerchantToken: "token-1",
		ReturnURL:     "https://shop.example.com/pay",
		CallbackURL:   "https://api.example.com/api/v1/payments/callback",
		ExchangeRate:  "7",
		Coin:          "USDT",
	}
	result, err := CreatePayment(context.Background(), cfg, CreateInput{
		UniqueID: "DJP1001",
		Name:     "????",
		Amount:   "18.80",
	})
	if err != nil {
		t.Fatalf("CreatePayment failed: %v", err)
	}
	if result.OrderID != "OK-ORDER-1" {
		t.Fatalf("unexpected order id: %s", result.OrderID)
	}
	if result.PayURL != "https://pay.example.com/ok" {
		t.Fatalf("unexpected pay url: %s", result.PayURL)
	}
	if !strings.Contains(receivedBody, "unique_id=DJP1001") {
		t.Fatalf("request body should contain unique_id, got %s", receivedBody)
	}
	if !strings.Contains(receivedBody, "amount=131.60000000") {
		t.Fatalf("request body should contain converted amount, got %s", receivedBody)
	}
	if !strings.Contains(receivedBody, "sign=") {
		t.Fatalf("request body should contain sign, got %s", receivedBody)
	}
}

func TestVerifyCallbackFallsBackToSortedKeys(t *testing.T) {
	cfg := &Config{
		MerchantID:    "1",
		MerchantToken: "123456",
	}
	bodyWithoutSign := "id=1&status=success&code=200&data[amount]=6.00000000&data[coin]=USDT&data[order_id]=ac7b86615fdb137576ae35879f7ed844&data[pay_user_id]=7238234930&data[status]=1&data[type]=deposit&data[unique_id]=BWIN-20250922152023LDVNSyxLQko"
	sign := md5Hex("code=200&data[amount]=6.00000000&data[coin]=USDT&data[order_id]=ac7b86615fdb137576ae35879f7ed844&data[pay_user_id]=7238234930&data[status]=1&data[type]=deposit&data[unique_id]=BWIN-20250922152023LDVNSyxLQko&id=1&status=success&token=123456")
	data, err := ParseCallback([]byte(bodyWithoutSign + "&sign=" + sign))
	if err != nil {
		t.Fatalf("ParseCallback failed: %v", err)
	}
	if err := VerifyCallback(cfg, data); err != nil {
		t.Fatalf("VerifyCallback failed: %v", err)
	}
}

func TestParseJSONCallbackAndVerify(t *testing.T) {
	cfg := &Config{
		MerchantID:    "1",
		MerchantToken: "123456",
	}
	body := `{"code":200,"data":{"order_id":"ac7b86615fdb137576ae35879f7ed844","unique_id":"BWIN-20250922152023LDVNSyxLQko","pay_user_id":7238234930,"amount":"6.00000000","coin":"USDT","status":1,"type":"deposit"},"id":1,"status":"success","sign":"95BE540FB7D1996770E2B4CDBC6F184D"}`
	data, err := ParseCallback([]byte(body))
	if err != nil {
		t.Fatalf("ParseCallback failed: %v", err)
	}
	if data.OrderID != "ac7b86615fdb137576ae35879f7ed844" {
		t.Fatalf("unexpected order id: %s", data.OrderID)
	}
	if data.UniqueID != "BWIN-20250922152023LDVNSyxLQko" {
		t.Fatalf("unexpected unique id: %s", data.UniqueID)
	}
	if data.PayUserID != "7238234930" {
		t.Fatalf("unexpected pay user id: %s", data.PayUserID)
	}
	if err := VerifyCallback(cfg, data); err != nil {
		t.Fatalf("VerifyCallback failed: %v", err)
	}
}

func TestConvertAmountByRate(t *testing.T) {
	converted, err := ConvertAmountByRate("1.00", "0.15")
	if err != nil {
		t.Fatalf("ConvertAmountByRate failed: %v", err)
	}
	if converted.StringFixed(8) != "0.15000000" {
		t.Fatalf("unexpected converted amount: %s", converted.StringFixed(8))
	}
}

func md5Hex(raw string) string {
	sum := md5.Sum([]byte(raw))
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}
