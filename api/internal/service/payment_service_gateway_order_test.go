package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"

	"github.com/shopspring/decimal"
)

func TestShouldUseGatewayOrderNo(t *testing.T) {
	tests := []struct {
		name    string
		channel *models.PaymentChannel
		want    bool
	}{
		{
			name: "epay",
			channel: &models.PaymentChannel{
				ProviderType: constants.PaymentProviderEpay,
			},
			want: true,
		},
		{
			name: "epusdt",
			channel: &models.PaymentChannel{
				ProviderType: constants.PaymentProviderEpusdt,
			},
			want: true,
		},
		{
			name: "okpay",
			channel: &models.PaymentChannel{
				ProviderType: constants.PaymentProviderOkpay,
			},
			want: true,
		},
		{
			name: "tokenpay",
			channel: &models.PaymentChannel{
				ProviderType: constants.PaymentProviderTokenpay,
			},
			want: true,
		},
		{
			name: "official wechat",
			channel: &models.PaymentChannel{
				ProviderType: constants.PaymentProviderOfficial,
				ChannelType:  constants.PaymentChannelTypeWechat,
			},
			want: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := shouldUseGatewayOrderNo(tc.channel); got != tc.want {
				t.Fatalf("shouldUseGatewayOrderNo() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestResolveGatewayOrderNo(t *testing.T) {
	channel := &models.PaymentChannel{ProviderType: constants.PaymentProviderEpusdt}
	payment := &models.Payment{ID: 123}

	gotFirst := resolveGatewayOrderNo(channel, payment)
	if gotFirst == "" {
		t.Fatalf("resolveGatewayOrderNo() should generate gateway order no")
	}
	if gotFirst == "DJP123" {
		t.Fatalf("resolveGatewayOrderNo() should not leak payment id, got %s", gotFirst)
	}
	if !strings.HasPrefix(gotFirst, "DJP") {
		t.Fatalf("resolveGatewayOrderNo() should keep DJP prefix, got %s", gotFirst)
	}

	payment.GatewayOrderNo = "CUSTOM-1"
	if got := resolveGatewayOrderNo(channel, payment); got != "CUSTOM-1" {
		t.Fatalf("resolveGatewayOrderNo() should reuse stored value, got %s", got)
	}
	payment.GatewayOrderNo = ""

	gotFallback := resolveGatewayOrderNo(channel, payment)
	if gotFallback == "" {
		t.Fatalf("resolveGatewayOrderNo() fallback should not be empty")
	}
	if gotFallback == "DJP123" {
		t.Fatalf("resolveGatewayOrderNo() fallback should not leak payment id, got %s", gotFallback)
	}
	if !strings.HasPrefix(gotFallback, "DJP") {
		t.Fatalf("resolveGatewayOrderNo() fallback should keep DJP prefix, got %s", gotFallback)
	}
	if gotFallback == gotFirst {
		t.Fatalf("resolveGatewayOrderNo() should generate new gateway order no for a new payment attempt")
	}

	official := &models.PaymentChannel{ProviderType: constants.PaymentProviderOfficial}
	if got := resolveGatewayOrderNo(official, payment); got == "" {
		t.Fatalf("official provider should also use gateway order no")
	}
}

func TestApplyProviderPaymentUsesGatewayOrderNoForEpusdt(t *testing.T) {
	svc, db := setupPaymentServiceWalletTest(t)
	now := time.Now()

	order := &models.Order{
		OrderNo:                 "DJTESTGATEWAY001",
		UserID:                  1,
		Status:                  constants.OrderStatusPendingPayment,
		Currency:                "CNY",
		OriginalAmount:          models.NewMoneyFromDecimal(decimal.NewFromInt(50)),
		DiscountAmount:          models.NewMoneyFromDecimal(decimal.Zero),
		PromotionDiscountAmount: models.NewMoneyFromDecimal(decimal.Zero),
		TotalAmount:             models.NewMoneyFromDecimal(decimal.NewFromInt(50)),
		WalletPaidAmount:        models.NewMoneyFromDecimal(decimal.Zero),
		OnlinePaidAmount:        models.NewMoneyFromDecimal(decimal.NewFromInt(50)),
		RefundedAmount:          models.NewMoneyFromDecimal(decimal.Zero),
		CreatedAt:               now,
		UpdatedAt:               now,
	}
	if err := db.Create(order).Error; err != nil {
		t.Fatalf("create order failed: %v", err)
	}

	var gotOrderID string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request failed: %v", err)
		}
		gotOrderID = strings.TrimSpace(payload["order_id"].(string))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status_code":200,"message":"ok","data":{"trade_id":"TRX-1001","order_id":"` + gotOrderID + `","amount":"50.00","actual_amount":"7.00","token":"USDT","expiration_time":1800,"payment_url":"https://pay.example.com/checkout"}}`))
	}))
	defer server.Close()

	channel := &models.PaymentChannel{
		ProviderType:    constants.PaymentProviderEpusdt,
		ChannelType:     constants.PaymentChannelTypeUsdtTrc20,
		InteractionMode: constants.PaymentInteractionRedirect,
		FeeRate:         models.NewMoneyFromDecimal(decimal.Zero),
		ConfigJSON: models.JSON{
			"gateway_url": server.URL,
			"auth_token":  "token-001",
			"trade_type":  "usdt.trc20",
			"fiat":        "CNY",
			"notify_url":  "https://example.com/callback",
			"return_url":  "https://example.com/pay-return",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := db.Create(channel).Error; err != nil {
		t.Fatalf("create channel failed: %v", err)
	}

	payment := &models.Payment{
		OrderID:         order.ID,
		ChannelID:       channel.ID,
		ProviderType:    channel.ProviderType,
		ChannelType:     channel.ChannelType,
		InteractionMode: channel.InteractionMode,
		Amount:          models.NewMoneyFromDecimal(decimal.RequireFromString("50.00")),
		FeeRate:         models.NewMoneyFromDecimal(decimal.Zero),
		FeeAmount:       models.NewMoneyFromDecimal(decimal.Zero),
		Currency:        "CNY",
		Status:          constants.PaymentStatusInitiated,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := db.Create(payment).Error; err != nil {
		t.Fatalf("create payment failed: %v", err)
	}

	if err := svc.applyProviderPayment(CreatePaymentInput{
		ClientIP: "127.0.0.1",
		Context:  context.Background(),
	}, order, channel, payment); err != nil {
		t.Fatalf("applyProviderPayment failed: %v", err)
	}

	if payment.GatewayOrderNo == "" {
		t.Fatalf("payment gateway order no should not be empty")
	}
	if payment.GatewayOrderNo == order.OrderNo {
		t.Fatalf("payment gateway order no should differ from business order no, got %s", payment.GatewayOrderNo)
	}
	if !strings.HasPrefix(payment.GatewayOrderNo, "DJP") {
		t.Fatalf("payment gateway order no should keep DJP prefix, got %s", payment.GatewayOrderNo)
	}
	if gotOrderID != payment.GatewayOrderNo {
		t.Fatalf("epusdt order_id = %s, want %s", gotOrderID, payment.GatewayOrderNo)
	}
	if payment.ProviderRef != "TRX-1001" {
		t.Fatalf("provider ref = %s, want TRX-1001", payment.ProviderRef)
	}
}

func TestApplyProviderPaymentUsesGatewayOrderNoForOkpay(t *testing.T) {
	svc, db := setupPaymentServiceWalletTest(t)
	now := time.Now()

	order := &models.Order{
		OrderNo:                 "DJTESTOKPAY001",
		UserID:                  1,
		Status:                  constants.OrderStatusPendingPayment,
		Currency:                "CNY",
		OriginalAmount:          models.NewMoneyFromDecimal(decimal.NewFromInt(88)),
		DiscountAmount:          models.NewMoneyFromDecimal(decimal.Zero),
		PromotionDiscountAmount: models.NewMoneyFromDecimal(decimal.Zero),
		TotalAmount:             models.NewMoneyFromDecimal(decimal.NewFromInt(88)),
		WalletPaidAmount:        models.NewMoneyFromDecimal(decimal.Zero),
		OnlinePaidAmount:        models.NewMoneyFromDecimal(decimal.NewFromInt(88)),
		RefundedAmount:          models.NewMoneyFromDecimal(decimal.Zero),
		CreatedAt:               now,
		UpdatedAt:               now,
	}
	if err := db.Create(order).Error; err != nil {
		t.Fatalf("create order failed: %v", err)
	}

	var gotUniqueID string
	var gotAmount string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form failed: %v", err)
		}
		gotUniqueID = strings.TrimSpace(r.PostForm.Get("unique_id"))
		gotAmount = strings.TrimSpace(r.PostForm.Get("amount"))
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","code":200,"data":{"order_id":"OKPAY-ORDER-1001","pay_url":"https://pay.example.com/okpay"}}`))
	}))
	defer server.Close()

	channel := &models.PaymentChannel{
		ProviderType:    constants.PaymentProviderOkpay,
		ChannelType:     constants.PaymentChannelTypeUsdt,
		InteractionMode: constants.PaymentInteractionQR,
		FeeRate:         models.NewMoneyFromDecimal(decimal.Zero),
		ConfigJSON: models.JSON{
			"gateway_url":    server.URL,
			"merchant_id":    "shop-1001",
			"merchant_token": "token-1001",
			"return_url":     "https://example.com/pay",
			"callback_url":   "https://api.example.com/api/v1/payments/callback",
			"exchange_rate":  "7",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := db.Create(channel).Error; err != nil {
		t.Fatalf("create channel failed: %v", err)
	}

	payment := &models.Payment{
		OrderID:         order.ID,
		ChannelID:       channel.ID,
		ProviderType:    channel.ProviderType,
		ChannelType:     channel.ChannelType,
		InteractionMode: channel.InteractionMode,
		Amount:          models.NewMoneyFromDecimal(decimal.RequireFromString("88.00")),
		FeeRate:         models.NewMoneyFromDecimal(decimal.Zero),
		FeeAmount:       models.NewMoneyFromDecimal(decimal.Zero),
		Currency:        "CNY",
		Status:          constants.PaymentStatusInitiated,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := db.Create(payment).Error; err != nil {
		t.Fatalf("create payment failed: %v", err)
	}

	if err := svc.applyProviderPayment(CreatePaymentInput{
		ClientIP: "127.0.0.1",
		Context:  context.Background(),
	}, order, channel, payment); err != nil {
		t.Fatalf("applyProviderPayment failed: %v", err)
	}

	if payment.GatewayOrderNo == "" {
		t.Fatalf("payment gateway order no should not be empty")
	}
	if gotUniqueID != payment.GatewayOrderNo {
		t.Fatalf("okpay unique_id = %s, want %s", gotUniqueID, payment.GatewayOrderNo)
	}
	if gotAmount != "616.00000000" {
		t.Fatalf("okpay amount = %s, want 616.00000000", gotAmount)
	}
	if payment.ProviderRef != "OKPAY-ORDER-1001" {
		t.Fatalf("provider ref = %s, want OKPAY-ORDER-1001", payment.ProviderRef)
	}
	if payment.PayURL != "https://pay.example.com/okpay" {
		t.Fatalf("unexpected pay url: %s", payment.PayURL)
	}
	if payment.QRCode != "https://pay.example.com/okpay" {
		t.Fatalf("unexpected qr code: %s", payment.QRCode)
	}
}

func TestApplyProviderPaymentBuildsRedirectURLForEpay(t *testing.T) {
	svc, db := setupPaymentServiceWalletTest(t)
	now := time.Now()

	order := &models.Order{
		OrderNo:                 "DJTESTEPAYREDIRECT001",
		UserID:                  1,
		Status:                  constants.OrderStatusPendingPayment,
		Currency:                "CNY",
		OriginalAmount:          models.NewMoneyFromDecimal(decimal.NewFromInt(20)),
		DiscountAmount:          models.NewMoneyFromDecimal(decimal.Zero),
		PromotionDiscountAmount: models.NewMoneyFromDecimal(decimal.Zero),
		TotalAmount:             models.NewMoneyFromDecimal(decimal.NewFromInt(20)),
		WalletPaidAmount:        models.NewMoneyFromDecimal(decimal.Zero),
		OnlinePaidAmount:        models.NewMoneyFromDecimal(decimal.NewFromInt(20)),
		RefundedAmount:          models.NewMoneyFromDecimal(decimal.Zero),
		CreatedAt:               now,
		UpdatedAt:               now,
	}
	if err := db.Create(order).Error; err != nil {
		t.Fatalf("create order failed: %v", err)
	}

	channel := &models.PaymentChannel{
		ProviderType:    constants.PaymentProviderEpay,
		ChannelType:     constants.PaymentChannelTypeAlipay,
		InteractionMode: constants.PaymentInteractionRedirect,
		FeeRate:         models.NewMoneyFromDecimal(decimal.Zero),
		ConfigJSON: models.JSON{
			"gateway_url":  "https://gateway.example.com",
			"epay_version": "v1",
			"merchant_id":  "1001",
			"merchant_key": "key-001",
			"notify_url":   "https://api.example.com/api/v1/payments/callback",
			"return_url":   "https://shop.example.com/pay",
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := db.Create(channel).Error; err != nil {
		t.Fatalf("create channel failed: %v", err)
	}

	payment := &models.Payment{
		OrderID:         order.ID,
		ChannelID:       channel.ID,
		ProviderType:    channel.ProviderType,
		ChannelType:     channel.ChannelType,
		InteractionMode: channel.InteractionMode,
		Amount:          models.NewMoneyFromDecimal(decimal.RequireFromString("20.00")),
		FeeRate:         models.NewMoneyFromDecimal(decimal.Zero),
		FeeAmount:       models.NewMoneyFromDecimal(decimal.Zero),
		Currency:        "CNY",
		Status:          constants.PaymentStatusInitiated,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := db.Create(payment).Error; err != nil {
		t.Fatalf("create payment failed: %v", err)
	}

	if err := svc.applyProviderPayment(CreatePaymentInput{
		ClientIP: "127.0.0.1",
		Context:  context.Background(),
	}, order, channel, payment); err != nil {
		t.Fatalf("applyProviderPayment redirect failed: %v", err)
	}

	if payment.PayURL == "" {
		t.Fatalf("pay url should not be empty")
	}
	if !strings.HasPrefix(payment.PayURL, "https://gateway.example.com/submit.php?") {
		t.Fatalf("unexpected pay url: %s", payment.PayURL)
	}
	if payment.QRCode != "" {
		t.Fatalf("qr code should stay empty in redirect mode")
	}
	if payment.ProviderPayload == nil {
		t.Fatalf("provider payload should be recorded")
	}
}

func TestValidateChannelRejectsInvalidEpayInteractionMode(t *testing.T) {
	svc, _ := setupPaymentServiceWalletTest(t)
	channel := &models.PaymentChannel{
		ProviderType:    constants.PaymentProviderEpay,
		ChannelType:     constants.PaymentChannelTypeWechat,
		InteractionMode: constants.PaymentInteractionPage,
		ConfigJSON: models.JSON{
			"gateway_url":  "https://gateway.example.com",
			"epay_version": "v1",
			"merchant_id":  "1001",
			"merchant_key": "key-001",
			"notify_url":   "https://api.example.com/api/v1/payments/callback",
			"return_url":   "https://shop.example.com/pay",
		},
	}
	if err := svc.ValidateChannel(channel); err == nil {
		t.Fatalf("ValidateChannel should reject invalid epay interaction mode")
	}
}

func TestValidateChannelAllowsUnlimitedMaxAmount(t *testing.T) {
	svc, _ := setupPaymentServiceWalletTest(t)
	channel := &models.PaymentChannel{
		ProviderType:    constants.PaymentProviderEpay,
		ChannelType:     constants.PaymentChannelTypeWechat,
		InteractionMode: constants.PaymentInteractionRedirect,
		MinAmount:       models.NewMoneyFromDecimal(decimal.RequireFromString("10.00")),
		MaxAmount:       models.NewMoneyFromDecimal(decimal.Zero),
		ConfigJSON: models.JSON{
			"gateway_url":  "https://gateway.example.com",
			"epay_version": "v1",
			"merchant_id":  "1001",
			"merchant_key": "key-001",
			"notify_url":   "https://api.example.com/api/v1/payments/callback",
			"return_url":   "https://shop.example.com/pay",
		},
	}

	if err := svc.ValidateChannel(channel); err != nil {
		t.Fatalf("ValidateChannel should allow max_amount=0 (unlimited): %v", err)
	}
}

func TestHandleCallbackAcceptsGatewayOrderNoForOrderPayment(t *testing.T) {
	svc, db := setupPaymentServiceWalletTest(t)
	now := time.Now()

	order := &models.Order{
		OrderNo:                 "DJTESTCALLBACK001",
		UserID:                  1,
		Status:                  constants.OrderStatusPendingPayment,
		Currency:                "CNY",
		OriginalAmount:          models.NewMoneyFromDecimal(decimal.NewFromInt(88)),
		DiscountAmount:          models.NewMoneyFromDecimal(decimal.Zero),
		PromotionDiscountAmount: models.NewMoneyFromDecimal(decimal.Zero),
		TotalAmount:             models.NewMoneyFromDecimal(decimal.NewFromInt(88)),
		WalletPaidAmount:        models.NewMoneyFromDecimal(decimal.Zero),
		OnlinePaidAmount:        models.NewMoneyFromDecimal(decimal.NewFromInt(88)),
		RefundedAmount:          models.NewMoneyFromDecimal(decimal.Zero),
		CreatedAt:               now,
		UpdatedAt:               now,
	}
	if err := db.Create(order).Error; err != nil {
		t.Fatalf("create order failed: %v", err)
	}

	payment := &models.Payment{
		OrderID:         order.ID,
		ChannelID:       1,
		ProviderType:    constants.PaymentProviderEpusdt,
		ChannelType:     constants.PaymentChannelTypeUsdtTrc20,
		InteractionMode: constants.PaymentInteractionRedirect,
		Amount:          models.NewMoneyFromDecimal(decimal.NewFromInt(88)),
		FeeRate:         models.NewMoneyFromDecimal(decimal.Zero),
		FeeAmount:       models.NewMoneyFromDecimal(decimal.Zero),
		Currency:        "CNY",
		Status:          constants.PaymentStatusPending,
		GatewayOrderNo:  "DJP501",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := db.Create(payment).Error; err != nil {
		t.Fatalf("create payment failed: %v", err)
	}

	updated, err := svc.HandleCallback(PaymentCallbackInput{
		PaymentID:   payment.ID,
		OrderNo:     payment.GatewayOrderNo,
		ChannelID:   payment.ChannelID,
		Status:      constants.PaymentStatusPending,
		ProviderRef: "TRX-PENDING-1",
		Amount:      payment.Amount,
		Currency:    payment.Currency,
		PaidAt:      ptrTime(time.Now()),
	})
	if err != nil {
		t.Fatalf("HandleCallback should accept gateway order no, got: %v", err)
	}
	if updated == nil || updated.ID != payment.ID {
		t.Fatalf("expected updated payment")
	}
}

func TestHandleCallbackAcceptsGatewayOrderNoForWalletRecharge(t *testing.T) {
	svc, db := setupPaymentServiceWalletTest(t)
	payment, recharge := createWalletRechargeFixture(t, db, constants.PaymentStatusPending, constants.WalletRechargeStatusPending)

	payment.GatewayOrderNo = "DJP8801"
	if err := db.Save(payment).Error; err != nil {
		t.Fatalf("save payment gateway order no failed: %v", err)
	}

	input := buildWalletRechargeCallbackInput(payment, recharge, constants.PaymentStatusPending, "CALLBACK-PENDING-GATEWAY")
	input.OrderNo = payment.GatewayOrderNo

	updated, err := svc.HandleCallback(input)
	if err != nil {
		t.Fatalf("HandleCallback should accept recharge gateway order no, got: %v", err)
	}
	if updated == nil || updated.ID != payment.ID {
		t.Fatalf("expected updated payment")
	}
}
