package dto

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/dujiao-next/internal/models"
)

func TestWalletAccountRespOmitsSensitiveFields(t *testing.T) {
	account := &models.WalletAccount{
		ID:      1,
		UserID:  99,
		Balance: newMoney("500.00"),
	}

	resp := NewWalletAccountResp(account)
	data, _ := json.Marshal(resp)
	jsonStr := string(data)

	if strings.Contains(jsonStr, `"user_id"`) {
		t.Error("user_id should not appear")
	}
	if strings.Contains(jsonStr, `"id"`) {
		t.Error("id should not appear")
	}
	if !strings.Contains(jsonStr, `"balance"`) {
		t.Error("balance should appear")
	}
}

func TestWalletTransactionRespOmitsSensitiveFields(t *testing.T) {
	txn := &models.WalletTransaction{
		ID:            1,
		UserID:        99,
		Type:          "recharge",
		Direction:     "in",
		Amount:        newMoney("100.00"),
		BalanceBefore: newMoney("400.00"),
		BalanceAfter:  newMoney("500.00"),
		Currency:      "CNY",
		Reference:     "unique-ref-001",
		Remark:        "??",
		CreatedAt:     time.Now(),
	}

	resp := NewWalletTransactionResp(txn)
	data, _ := json.Marshal(resp)
	jsonStr := string(data)

	sensitiveFields := []string{"user_id", "balance_before", "reference", "currency", "updated_at"}
	for _, field := range sensitiveFields {
		if strings.Contains(jsonStr, `"`+field+`"`) {
			t.Errorf("sensitive field %q should not appear", field)
		}
	}

	if !strings.Contains(jsonStr, `"balance_after"`) {
		t.Error("balance_after should appear")
	}
	if !strings.Contains(jsonStr, `"remark"`) {
		t.Error("remark should appear")
	}
}

func TestWalletRechargePaymentPayloadOmitsSensitiveFields(t *testing.T) {
	expires := time.Now()
	recharge := &models.WalletRechargeOrder{
		ID:         1,
		RechargeNo: "RCG-001",
		UserID:     99,
		PaymentID:  5,
		ChannelID:  3,
		FeeRate:    newMoney("2.50"),
		Amount:     newMoney("100.00"),
		Status:     "success",
	}
	payment := &models.Payment{
		ID:              5,
		ProviderType:    "alipay",
		ChannelType:     "alipay",
		InteractionMode: "redirect",
		FeeRate:         newMoney("1.00"),
		FixedFee:        newMoney("0.50"),
		ProviderRef:     "ALI-REF-001",
		GatewayOrderNo:  "GW-001",
		ProviderPayload: models.JSON{"raw": "data"},
		PayURL:          "https://pay.example.com",
		ExpiredAt:       &expires,
		Status:          "paid",
	}
	account := &models.WalletAccount{Balance: newMoney("200.00")}

	resp := NewWalletRechargePaymentPayload(recharge, payment, account)
	data, _ := json.Marshal(resp)
	jsonStr := string(data)

	// Recharge ????
	sensitiveFields := []string{"user_id", "channel_id", "fee_rate"}
	for _, field := range sensitiveFields {
		if strings.Contains(jsonStr, `"`+field+`"`) {
			t.Errorf("sensitive field %q should not appear", field)
		}
	}

	// Payment ????
	paymentSensitive := []string{"provider_ref", "gateway_order_no", "provider_payload", "fixed_fee"}
	for _, field := range paymentSensitive {
		if strings.Contains(jsonStr, `"`+field+`"`) {
			t.Errorf("payment sensitive field %q should not appear", field)
		}
	}

	if !strings.Contains(jsonStr, `"pay_url"`) {
		t.Error("pay_url should appear")
	}
	if !strings.Contains(jsonStr, `"recharge_no"`) {
		t.Error("recharge_no should appear")
	}
}
