package service

import (
	"testing"

	"github.com/dujiao-next/internal/models"
)

func TestBuildPaymentReturnQueryForOrder(t *testing.T) {
	order := &models.Order{
		OrderNo: "DJ202603230001",
		UserID:  0,
	}

	params := buildPaymentReturnQuery(CreatePaymentInput{}, order, "epay_return", "SESSION-ORDER-1")

	if got := params["biz_type"]; got != "order" {
		t.Fatalf("biz_type want order got %q", got)
	}
	if got := params["order_no"]; got != "DJ202603230001" {
		t.Fatalf("order_no want DJ202603230001 got %q", got)
	}
	if got := params["guest"]; got != "1" {
		t.Fatalf("guest want 1 got %q", got)
	}
	if got := params["epay_return"]; got != "1" {
		t.Fatalf("epay_return want 1 got %q", got)
	}
	if got := params["session_id"]; got != "SESSION-ORDER-1" {
		t.Fatalf("session_id want SESSION-ORDER-1 got %q", got)
	}
	if _, exists := params["recharge_no"]; exists {
		t.Fatalf("recharge_no should be absent: %#v", params)
	}
}

func TestBuildPaymentReturnQueryForRecharge(t *testing.T) {
	order := &models.Order{
		OrderNo: "DJRECHARGE0001",
		UserID:  100,
	}
	input := CreatePaymentInput{
		ReturnBizType:    "recharge",
		ReturnBusinessNo: "DJRECHARGE0001",
	}

	params := buildPaymentReturnQuery(input, order, "stripe_return", "SESSION-RECHARGE-1")

	if got := params["biz_type"]; got != "recharge" {
		t.Fatalf("biz_type want recharge got %q", got)
	}
	if got := params["recharge_no"]; got != "DJRECHARGE0001" {
		t.Fatalf("recharge_no want DJRECHARGE0001 got %q", got)
	}
	if got := params["stripe_return"]; got != "1" {
		t.Fatalf("stripe_return want 1 got %q", got)
	}
	if got := params["session_id"]; got != "SESSION-RECHARGE-1" {
		t.Fatalf("session_id want SESSION-RECHARGE-1 got %q", got)
	}
	if _, exists := params["order_no"]; exists {
		t.Fatalf("order_no should be absent: %#v", params)
	}
	if _, exists := params["guest"]; exists {
		t.Fatalf("guest should be absent: %#v", params)
	}
}
