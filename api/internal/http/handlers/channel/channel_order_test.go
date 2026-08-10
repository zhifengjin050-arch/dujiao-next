package channel

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

func TestBuildChannelOrderPreviewResponseIncludesTelegramFriendlyFields(t *testing.T) {
	resp := buildChannelOrderPreviewResponse(&service.OrderPreview{
		Currency:                "CNY",
		OriginalAmount:          models.NewMoneyFromDecimal(decimal.RequireFromString("108.00")),
		DiscountAmount:          models.NewMoneyFromDecimal(decimal.RequireFromString("8.00")),
		PromotionDiscountAmount: models.NewMoneyFromDecimal(decimal.RequireFromString("10.00")),
		TotalAmount:             models.NewMoneyFromDecimal(decimal.RequireFromString("90.00")),
		Items: []service.OrderPreviewItem{{
			ProductID:         12,
			SKUID:             34,
			TitleJSON:         models.JSON{"zh-CN": "????"},
			SKUSnapshotJSON:   models.JSON{"spec_values": models.JSON{"zh-CN": "???"}},
			Quantity:          2,
			UnitPrice:         models.NewMoneyFromDecimal(decimal.RequireFromString("54.00")),
			TotalPrice:        models.NewMoneyFromDecimal(decimal.RequireFromString("108.00")),
			CouponDiscount:    models.NewMoneyFromDecimal(decimal.RequireFromString("8.00")),
			PromotionDiscount: models.NewMoneyFromDecimal(decimal.RequireFromString("10.00")),
			FulfillmentType:   "manual",
		}},
	}, "zh-CN")

	if got := resp["item_count"]; got != 1 {
		t.Fatalf("expected item_count=1, got=%v", got)
	}
	if got := resp["original_amount"]; got != "108.00" {
		t.Fatalf("expected original_amount=108.00, got=%v", got)
	}
	items, ok := resp["items"].([]gin.H)
	if !ok || len(items) != 1 {
		t.Fatalf("expected single preview item, got=%T len=%d", resp["items"], len(items))
	}
	if got := items[0]["coupon_discount"]; got != "8.00" {
		t.Fatalf("expected coupon_discount=8.00, got=%v", got)
	}
	if got := items[0]["promotion_discount"]; got != "10.00" {
		t.Fatalf("expected promotion_discount=10.00, got=%v", got)
	}
	if got := items[0]["fulfillment_type"]; got != "manual" {
		t.Fatalf("expected fulfillment_type=manual, got=%v", got)
	}
}

func TestBuildChannelOrderDetailResponseUsesTotalPaidAmount(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 0, 0, 0, time.UTC)
	resp := buildChannelOrderDetailResponse(&models.Order{
		ID:                      7,
		OrderNo:                 "DJ20260310001",
		Status:                  "paid",
		Currency:                "CNY",
		OriginalAmount:          models.NewMoneyFromDecimal(decimal.RequireFromString("100.00")),
		DiscountAmount:          models.NewMoneyFromDecimal(decimal.RequireFromString("5.00")),
		PromotionDiscountAmount: models.NewMoneyFromDecimal(decimal.RequireFromString("15.00")),
		TotalAmount:             models.NewMoneyFromDecimal(decimal.RequireFromString("80.00")),
		WalletPaidAmount:        models.NewMoneyFromDecimal(decimal.RequireFromString("20.00")),
		OnlinePaidAmount:        models.NewMoneyFromDecimal(decimal.RequireFromString("60.00")),
		RefundedAmount:          models.NewMoneyFromDecimal(decimal.RequireFromString("0.00")),
		ExpiresAt:               &now,
		CreatedAt:               now,
		UpdatedAt:               now,
		PaidAt:                  &now,
		Items: []models.OrderItem{{
			ProductID:         1,
			SKUID:             2,
			TitleJSON:         models.JSON{"zh-CN": "????"},
			SKUSnapshotJSON:   models.JSON{"spec_values": models.JSON{"zh-CN": "???"}},
			Quantity:          1,
			UnitPrice:         models.NewMoneyFromDecimal(decimal.RequireFromString("80.00")),
			TotalPrice:        models.NewMoneyFromDecimal(decimal.RequireFromString("80.00")),
			CouponDiscount:    models.NewMoneyFromDecimal(decimal.RequireFromString("5.00")),
			PromotionDiscount: models.NewMoneyFromDecimal(decimal.RequireFromString("15.00")),
			FulfillmentType:   "manual",
		}},
		Children: []models.Order{{
			ID:      8,
			OrderNo: "DJ20260310001-01",
			Status:  "completed",
			Fulfillment: &models.Fulfillment{
				Status:      "delivered",
				Type:        "auto",
				Payload:     "card-secret-demo",
				DeliveredAt: &now,
			},
		}},
	}, "zh-CN")

	if got := resp["paid_amount"]; got != "80.00" {
		t.Fatalf("expected paid_amount=80.00, got=%v", got)
	}
	if got := resp["wallet_paid_amount"]; got != "20.00" {
		t.Fatalf("expected wallet_paid_amount=20.00, got=%v", got)
	}
	if got := resp["online_paid_amount"]; got != "60.00" {
		t.Fatalf("expected online_paid_amount=60.00, got=%v", got)
	}
	if got := resp["item_count"]; got != 1 {
		t.Fatalf("expected item_count=1, got=%v", got)
	}
	if got := resp["fulfillment_type"]; got != "manual" {
		t.Fatalf("expected fulfillment_type=manual, got=%v", got)
	}
	items, ok := resp["items"].([]gin.H)
	if !ok || len(items) != 1 {
		t.Fatalf("expected single order item, got=%T len=%d", resp["items"], len(items))
	}
	if got := items[0]["coupon_discount"]; got != "5.00" {
		t.Fatalf("expected coupon_discount=5.00, got=%v", got)
	}
	if got := items[0]["promotion_discount"]; got != "15.00" {
		t.Fatalf("expected promotion_discount=15.00, got=%v", got)
	}
	if got := resp["fulfillment_delivered_at"]; got != nil {
		t.Fatalf("expected parent fulfillment_delivered_at=nil, got=%v", got)
	}
	children, ok := resp["children"].([]gin.H)
	if !ok || len(children) != 1 {
		t.Fatalf("expected single child order, got=%T len=%d", resp["children"], len(children))
	}
	childFulfillment, ok := children[0]["fulfillment"].(gin.H)
	if !ok {
		t.Fatalf("expected child fulfillment payload, got=%T", children[0]["fulfillment"])
	}
	if got := childFulfillment["payload"]; got != "card-secret-demo" {
		t.Fatalf("expected child fulfillment payload=card-secret-demo, got=%v", got)
	}
	if got := childFulfillment["status"]; got != "delivered" {
		t.Fatalf("expected child fulfillment status=delivered, got=%v", got)
	}
}

func TestBuildChannelPaymentResponseIncludesOrderSummary(t *testing.T) {
	now := time.Date(2026, 3, 10, 12, 30, 0, 0, time.UTC)
	order := &models.Order{
		ID:               9,
		OrderNo:          "DJ20260310002",
		TotalAmount:      models.NewMoneyFromDecimal(decimal.RequireFromString("99.00")),
		WalletPaidAmount: models.NewMoneyFromDecimal(decimal.RequireFromString("19.00")),
		OnlinePaidAmount: models.NewMoneyFromDecimal(decimal.RequireFromString("80.00")),
	}
	payment := &models.Payment{
		ID:              5,
		OrderID:         order.ID,
		ChannelID:       11,
		Status:          "pending",
		ProviderType:    "alipay",
		ChannelType:     "alipay",
		InteractionMode: "redirect",
		Amount:          models.NewMoneyFromDecimal(decimal.RequireFromString("80.80")),
		FeeRate:         models.NewMoneyFromDecimal(decimal.RequireFromString("1.00")),
		FeeAmount:       models.NewMoneyFromDecimal(decimal.RequireFromString("0.80")),
		Currency:        "CNY",
		PayURL:          "https://pay.example.com",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	resp := buildChannelPaymentResponse(order, payment)
	if got := resp["channel_id"]; got != uint(11) {
		t.Fatalf("expected channel_id=11, got=%v", got)
	}
	if got := resp["fee_amount"]; got != "0.80" {
		t.Fatalf("expected fee_amount=0.80, got=%v", got)
	}
	if got := resp["order_no"]; got != "DJ20260310002" {
		t.Fatalf("expected order_no=DJ20260310002, got=%v", got)
	}
	if got := resp["paid_amount"]; got != "99.00" {
		t.Fatalf("expected paid_amount=99.00, got=%v", got)
	}
}

func TestPreviewOrderRequestBindsAffiliateFields(t *testing.T) {
	raw := []byte(`{
		"channel_user_id":"998877",
		"telegram_user_id":"998877",
		"items":[{"product_id":12,"sku_id":34,"quantity":1,"fulfillment_type":"manual"}],
		"affiliate_code":"AFFTG001",
		"affiliate_visitor_key":"visitor-998877"
	}`)

	var req previewOrderRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		t.Fatalf("unmarshal preview order request failed: %v", err)
	}
	if req.AffiliateCode != "AFFTG001" {
		t.Fatalf("expected affiliate code to bind, got=%s", req.AffiliateCode)
	}
	if req.AffiliateKey != "visitor-998877" {
		t.Fatalf("expected affiliate visitor key to bind, got=%s", req.AffiliateKey)
	}
}

func TestCreateOrderRequestBindsAffiliateFields(t *testing.T) {
	raw := []byte(`{
		"channel_user_id":"556677",
		"telegram_user_id":"556677",
		"product_id":12,
		"sku_id":34,
		"quantity":2,
		"affiliate_code":"AFFTG002",
		"affiliate_visitor_key":"visitor-556677"
	}`)

	var req createOrderRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		t.Fatalf("unmarshal create order request failed: %v", err)
	}
	if req.AffiliateCode != "AFFTG002" {
		t.Fatalf("expected affiliate code to bind, got=%s", req.AffiliateCode)
	}
	if req.AffiliateKey != "visitor-556677" {
		t.Fatalf("expected affiliate visitor key to bind, got=%s", req.AffiliateKey)
	}
}
