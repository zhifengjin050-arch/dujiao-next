package dto

import (
	"strings"
	"time"

	"github.com/dujiao-next/internal/models"
)

// OrderSummary ??????(????)
type OrderSummary struct {
	OrderNo     string          `json:"order_no"`
	Status      string          `json:"status"`
	Currency    string          `json:"currency"`
	TotalAmount models.Money    `json:"total_amount"`
	CreatedAt   time.Time       `json:"created_at"`
	Items       []OrderItemResp `json:"items,omitempty"`
	Children    []OrderSummary  `json:"children,omitempty"`
}

// NewOrderSummary ? models.Order ?? OrderSummary
func NewOrderSummary(o *models.Order) OrderSummary {
	s := OrderSummary{
		OrderNo:     o.OrderNo,
		Status:      o.Status,
		Currency:    o.Currency,
		TotalAmount: o.TotalAmount,
		CreatedAt:   o.CreatedAt,
	}
	for _, item := range o.Items {
		resp := newOrderItemResp(&item)
		resp.Instructions = nil
		s.Items = append(s.Items, resp)
	}
	for i := range o.Children {
		child := NewOrderSummary(&o.Children[i])
		s.Children = append(s.Children, child)
	}
	return s
}

// NewOrderSummaryList ????????
func NewOrderSummaryList(orders []models.Order) []OrderSummary {
	result := make([]OrderSummary, 0, len(orders))
	for i := range orders {
		result = append(result, NewOrderSummary(&orders[i]))
	}
	return result
}

// OrderDetail ??????(????)
type OrderDetail struct {
	OrderNo                  string            `json:"order_no"`
	GuestEmail               string            `json:"guest_email,omitempty"`
	GuestLocale              string            `json:"guest_locale,omitempty"`
	Status                   string            `json:"status"`
	Currency                 string            `json:"currency"`
	OriginalAmount           models.Money      `json:"original_amount"`
	DiscountAmount           models.Money      `json:"discount_amount"`
	MemberDiscountAmount     models.Money      `json:"member_discount_amount"`
	PromotionDiscountAmount  models.Money      `json:"promotion_discount_amount"`
	TotalAmount              models.Money      `json:"total_amount"`
	WalletPaidAmount         models.Money      `json:"wallet_paid_amount"`
	OnlinePaidAmount         models.Money      `json:"online_paid_amount"`
	RefundedAmount           models.Money      `json:"refunded_amount"`
	ExpiresAt                *time.Time        `json:"expires_at"`
	PaidAt                   *time.Time        `json:"paid_at"`
	CanceledAt               *time.Time        `json:"canceled_at"`
	CreatedAt                time.Time         `json:"created_at"`
	AllowedPaymentChannelIDs []uint            `json:"allowed_payment_channel_ids,omitempty"`
	RefundRecords            []OrderRefundResp `json:"refund_records,omitempty"`
	Items                    []OrderItemResp   `json:"items,omitempty"`
	Fulfillment              *FulfillmentResp  `json:"fulfillment,omitempty"`
	Children                 []OrderDetail     `json:"children,omitempty"`
}

// OrderRefundResp ???????????
type OrderRefundResp struct {
	Type      string       `json:"type"`
	Amount    models.Money `json:"amount"`
	Currency  string       `json:"currency"`
	Remark    string       `json:"remark,omitempty"`
	CreatedAt time.Time    `json:"created_at"`
}

// NewOrderDetail ? models.Order ?? OrderDetail,
// ?????? upstream ???????????
func NewOrderDetail(o *models.Order) OrderDetail {
	d := OrderDetail{
		OrderNo:                 o.OrderNo,
		GuestEmail:              o.GuestEmail,
		GuestLocale:             o.GuestLocale,
		Status:                  o.Status,
		Currency:                o.Currency,
		OriginalAmount:          o.OriginalAmount,
		DiscountAmount:          o.DiscountAmount,
		MemberDiscountAmount:    o.MemberDiscountAmount,
		PromotionDiscountAmount: o.PromotionDiscountAmount,
		TotalAmount:             o.TotalAmount,
		WalletPaidAmount:        o.WalletPaidAmount,
		OnlinePaidAmount:        o.OnlinePaidAmount,
		RefundedAmount:          o.RefundedAmount,
		ExpiresAt:               o.ExpiresAt,
		PaidAt:                  o.PaidAt,
		CanceledAt:              o.CanceledAt,
		CreatedAt:               o.CreatedAt,
	}
	paid := o.PaidAt != nil
	for _, item := range o.Items {
		resp := newOrderItemResp(&item)
		if !paid {
			resp.Instructions = nil
		}
		d.Items = append(d.Items, resp)
	}
	if o.Fulfillment != nil {
		fr := newFulfillmentResp(o.Fulfillment)
		d.Fulfillment = &fr
	}
	for i := range o.Children {
		d.Children = append(d.Children, NewOrderDetail(&o.Children[i]))
	}
	return d
}

// NewOrderDetailTruncated ? NewOrderDetail,??????????
func NewOrderDetailTruncated(o *models.Order) OrderDetail {
	d := NewOrderDetail(o)
	truncateFulfillment(&d)
	return d
}

func truncateFulfillment(d *OrderDetail) {
	if d.Fulfillment != nil {
		d.Fulfillment.truncatePayload(models.FulfillmentPayloadMaxPreviewLines)
	}
	for i := range d.Children {
		truncateFulfillment(&d.Children[i])
	}
}

// OrderItemResp ?????
type OrderItemResp struct {
	Title                    models.JSON        `json:"title"`
	SKUSnapshot              models.JSON        `json:"sku_snapshot"`
	Tags                     models.StringArray `json:"tags"`
	Quantity                 int                `json:"quantity"`
	UnitPrice                models.Money       `json:"unit_price"`
	TotalPrice               models.Money       `json:"total_price"`
	CouponDiscountAmount     models.Money       `json:"coupon_discount_amount"`
	MemberDiscountAmount     models.Money       `json:"member_discount_amount"`
	PromotionDiscountAmount  models.Money       `json:"promotion_discount_amount"`
	FulfillmentType          string             `json:"fulfillment_type"`
	ManualFormSchemaSnapshot models.JSON        `json:"manual_form_schema_snapshot"`
	ManualFormSubmission     models.JSON        `json:"manual_form_submission"`
	// Instructions ??????(??? raw JSON,? Title ??????,???? locale ??)?
	// ??????????,??????? NewOrderDetail ?? nil;????(NewOrderSummary)??? nil?
	// ??:?? API(channel_order.go)???????(? Telegram Bot),?????? locale ???????? raw JSON?
	Instructions models.JSON `json:"instructions,omitempty"`
}

func newOrderItemResp(item *models.OrderItem) OrderItemResp {
	ft := item.FulfillmentType
	if ft == "upstream" {
		ft = "manual"
	}
	return OrderItemResp{
		Title:                    item.TitleJSON,
		SKUSnapshot:              item.SKUSnapshotJSON,
		Tags:                     item.Tags,
		Quantity:                 item.Quantity,
		UnitPrice:                item.UnitPrice,
		TotalPrice:               item.TotalPrice,
		CouponDiscountAmount:     item.CouponDiscount,
		MemberDiscountAmount:     item.MemberDiscount,
		PromotionDiscountAmount:  item.PromotionDiscount,
		FulfillmentType:          ft,
		ManualFormSchemaSnapshot: item.ManualFormSchemaSnapshotJSON,
		ManualFormSubmission:     item.ManualFormSubmissionJSON,
		Instructions:             item.InstructionsJSON,
	}
	// ??:CostPrice ?? DTO ?,?????????
}

// FulfillmentResp ??????
type FulfillmentResp struct {
	Type             string      `json:"type"`
	Status           string      `json:"status"`
	Payload          string      `json:"payload"`
	PayloadLineCount int         `json:"payload_line_count"`
	DeliveryData     models.JSON `json:"delivery_data"`
	DeliveredAt      *time.Time  `json:"delivered_at,omitempty"`
}

func newFulfillmentResp(f *models.Fulfillment) FulfillmentResp {
	typ := f.Type
	if typ == "upstream" {
		typ = "manual"
	}
	return FulfillmentResp{
		Type:             typ,
		Status:           f.Status,
		Payload:          f.Payload,
		PayloadLineCount: f.PayloadLineCount,
		DeliveryData:     f.LogisticsJSON,
		DeliveredAt:      f.DeliveredAt,
	}
	// ??:OrderID?DeliveredBy?CreatedAt?UpdatedAt ?? DTO ?
}

func (fr *FulfillmentResp) truncatePayload(maxLines int) {
	if fr.Payload == "" {
		return
	}
	lines := strings.Split(fr.Payload, "\n")
	fr.PayloadLineCount = len(lines)
	if len(lines) > maxLines {
		fr.Payload = strings.Join(lines[:maxLines], "\n")
	}
}
