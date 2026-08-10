package public

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dujiao-next/internal/constants"

	"github.com/gin-gonic/gin"
)

func TestMapAlipayTradeStatus(t *testing.T) {
	if status, ok := mapAlipayTradeStatus(constants.AlipayTradeStatusSuccess); !ok || status != constants.PaymentStatusSuccess {
		t.Fatalf("expected success mapping, got %s %v", status, ok)
	}
	if status, ok := mapAlipayTradeStatus(constants.AlipayTradeStatusWaitBuyerPay); !ok || status != constants.PaymentStatusPending {
		t.Fatalf("expected pending mapping, got %s %v", status, ok)
	}
	if status, ok := mapAlipayTradeStatus(constants.AlipayTradeStatusClosed); !ok || status != constants.PaymentStatusFailed {
		t.Fatalf("expected failed mapping, got %s %v", status, ok)
	}
	if status, ok := mapAlipayTradeStatus("UNKNOWN"); ok || status != "" {
		t.Fatalf("expected unknown mapping, got %s %v", status, ok)
	}
}

func TestParseAlipayCallback(t *testing.T) {
	form := map[string][]string{
		"out_trade_no": {"ORDER-1"},
		"trade_no":     {"202602090001"},
		"trade_status": {"TRADE_SUCCESS"},
		"total_amount": {"18.80"},
		"gmt_payment":  {"2026-02-09 23:30:00"},
	}
	input, err := parseAlipayCallback(form, 1001)
	if err != nil {
		t.Fatalf("parse alipay callback failed: %v", err)
	}
	if input.PaymentID != 1001 {
		t.Fatalf("expected payment id 1001, got %d", input.PaymentID)
	}
	if input.Status != constants.PaymentStatusSuccess {
		t.Fatalf("expected success status, got %s", input.Status)
	}
	if input.ProviderRef != "202602090001" {
		t.Fatalf("expected provider ref trade_no, got %s", input.ProviderRef)
	}
	if input.Amount.String() != "18.80" {
		t.Fatalf("expected amount 18.80, got %s", input.Amount.String())
	}
	if input.PaidAt == nil {
		t.Fatalf("expected paid_at parsed")
	}
}

func TestParseCallbackFormPreferPostForm(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	body := strings.NewReader("out_trade_no=ORDER-POST&trade_status=TRADE_SUCCESS&sign=abc&notify_id=n1")
	req := httptest.NewRequest(http.MethodPost, "/api/v1/payments/callback?channel_id=999", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	c.Request = req

	form, err := parseCallbackForm(c)
	if err != nil {
		t.Fatalf("parse callback form failed: %v", err)
	}
	if got := getFirstValue(form, "out_trade_no"); got != "ORDER-POST" {
		t.Fatalf("unexpected out_trade_no: %s", got)
	}
	if got := getFirstValue(form, "channel_id"); got != "" {
		t.Fatalf("expected query param excluded from signed form, got %s", got)
	}
}
