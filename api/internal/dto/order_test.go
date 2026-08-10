package dto

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/dujiao-next/internal/models"
)

func TestOrderDetailOmitsSensitiveFields(t *testing.T) {
	now := time.Now()
	couponID := uint(10)
	promotionID := uint(20)
	affiliateID := uint(30)
	memberLevelID := uint(5)

	order := &models.Order{
		ID:                 1,
		OrderNo:            "ORD-001",
		UserID:             99,
		GuestEmail:         "guest@test.com",
		GuestPassword:      "secret123",
		Status:             "paid",
		Currency:           "CNY",
		TotalAmount:        newMoney("100.00"),
		ClientIP:           "192.168.1.1",
		AffiliateProfileID: &affiliateID,
		AffiliateCode:      "AFF-SECRET",
		CouponID:           &couponID,
		PromotionID:        &promotionID,
		MemberLevelID:      &memberLevelID,
		CreatedAt:          now,
		UpdatedAt:          now,
		Items: []models.OrderItem{
			{
				ID:              1,
				OrderID:         1,
				ProductID:       5,
				SKUID:           10,
				TitleJSON:       models.JSON{"zh-CN": "??A"},
				CostPrice:       newMoney("50.00"),
				UnitPrice:       newMoney("100.00"),
				TotalPrice:      newMoney("100.00"),
				Quantity:        1,
				FulfillmentType: "upstream",
			},
		},
		Fulfillment: &models.Fulfillment{
			ID:     1,
			Type:   "upstream",
			Status: "delivered",
		},
	}

	detail := NewOrderDetail(order)
	data, err := json.Marshal(detail)
	if err != nil {
		t.Fatal(err)
	}
	jsonStr := string(data)

	// ????????
	sensitiveFields := []string{
		"client_ip", "affiliate_profile_id", "affiliate_code",
		"coupon_id", "promotion_id", "member_level_id",
		"user_id", "guest_password", "updated_at",
		"cost_price", "order_id", "product_id", "sku_id",
		"delivered_by", "parent_id",
	}
	for _, field := range sensitiveFields {
		if strings.Contains(jsonStr, `"`+field+`"`) {
			t.Errorf("sensitive field %q should not appear in JSON", field)
		}
	}

	// ???????
	publicFields := []string{
		"order_no", "total_amount", "status", "currency",
		"guest_email", "unit_price", "fulfillment_type",
	}
	for _, field := range publicFields {
		if !strings.Contains(jsonStr, `"`+field+`"`) {
			t.Errorf("public field %q should appear in JSON", field)
		}
	}

	// upstream ????? manual
	if strings.Contains(jsonStr, `"upstream"`) {
		t.Error("upstream fulfillment type should be masked as manual")
	}
	if !strings.Contains(jsonStr, `"manual"`) {
		t.Error("fulfillment type should be manual after masking")
	}
}

func TestOrderDetailHidesInstructionsBeforePayment(t *testing.T) {
	instructions := models.JSON{"zh-CN": "??????:."}
	mkOrder := func(paidAt *time.Time) *models.Order {
		return &models.Order{
			ID:          42,
			OrderNo:     "ORD-INST",
			Status:      "pending_payment",
			Currency:    "CNY",
			TotalAmount: newMoney("9.90"),
			PaidAt:      paidAt,
			Items: []models.OrderItem{
				{
					ID:               1,
					ProductID:        5,
					SKUID:            10,
					TitleJSON:        models.JSON{"zh-CN": "??A"},
					InstructionsJSON: instructions,
					FulfillmentType:  "auto",
				},
			},
		}
	}

	// ???:instructions ?????
	unpaid := NewOrderDetail(mkOrder(nil))
	if unpaid.Items[0].Instructions != nil {
		t.Fatalf("unpaid order must not expose instructions, got %v", unpaid.Items[0].Instructions)
	}

	// ???:instructions ????
	now := time.Now()
	paid := NewOrderDetail(mkOrder(&now))
	if paid.Items[0].Instructions == nil {
		t.Fatal("paid order should expose instructions")
	}
	if got := paid.Items[0].Instructions["zh-CN"]; got != "??????:." {
		t.Fatalf("instructions mismatch: %v", got)
	}

	// Summary ???????? instructions
	summary := NewOrderSummary(mkOrder(&now))
	if summary.Items[0].Instructions != nil {
		t.Fatal("order summary must never expose instructions")
	}
}

func TestOrderSummaryOmitsSensitiveFields(t *testing.T) {
	order := &models.Order{
		ID:            1,
		OrderNo:       "ORD-002",
		UserID:        99,
		ClientIP:      "10.0.0.1",
		AffiliateCode: "AFF-X",
		Status:        "pending",
		Currency:      "USD",
		TotalAmount:   newMoney("50.00"),
		CreatedAt:     time.Now(),
	}

	summary := NewOrderSummary(order)
	data, _ := json.Marshal(summary)
	jsonStr := string(data)

	if strings.Contains(jsonStr, `"client_ip"`) {
		t.Error("client_ip should not appear in summary")
	}
	if strings.Contains(jsonStr, `"user_id"`) {
		t.Error("user_id should not appear in summary")
	}
	if summary.OrderNo != "ORD-002" {
		t.Errorf("expected order_no=ORD-002, got %s", summary.OrderNo)
	}
}

func TestNewOrderDetailTruncated(t *testing.T) {
	lines := make([]string, 200)
	for i := range lines {
		lines[i] = "line"
	}
	order := &models.Order{
		ID:      1,
		OrderNo: "ORD-003",
		Status:  "delivered",
		Fulfillment: &models.Fulfillment{
			ID:      1,
			Type:    "auto",
			Status:  "delivered",
			Payload: strings.Join(lines, "\n"),
		},
	}

	detail := NewOrderDetailTruncated(order)
	if detail.Fulfillment == nil {
		t.Fatal("expected fulfillment")
	}
	if detail.Fulfillment.PayloadLineCount != 200 {
		t.Errorf("expected 200 lines, got %d", detail.Fulfillment.PayloadLineCount)
	}
	resultLines := strings.Split(detail.Fulfillment.Payload, "\n")
	if len(resultLines) != models.FulfillmentPayloadMaxPreviewLines {
		t.Errorf("expected truncated to %d lines, got %d", models.FulfillmentPayloadMaxPreviewLines, len(resultLines))
	}
}

func TestNewOrderDetailNilSafety(t *testing.T) {
	order := &models.Order{
		ID:      1,
		OrderNo: "ORD-NIL",
		Status:  "pending",
	}
	detail := NewOrderDetail(order)
	if detail.Fulfillment != nil {
		t.Error("expected nil fulfillment")
	}
	if detail.Items != nil {
		t.Error("expected nil items")
	}
}
