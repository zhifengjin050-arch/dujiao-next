package models

import (
	"time"

	"gorm.io/gorm"
)

// Order ???
type Order struct {
	ID                      uint           `gorm:"primarykey" json:"id"`                                                   // ??
	OrderNo                 string         `gorm:"uniqueIndex;not null" json:"order_no"`                                   // ????
	ParentID                *uint          `gorm:"index" json:"parent_id,omitempty"`                                       // ???ID
	UserID                  uint           `gorm:"index;not null" json:"user_id,omitempty"`                                // ??ID(????? 0)
	GuestEmail              string         `gorm:"index" json:"guest_email,omitempty"`                                     // ????
	GuestPassword           string         `gorm:"type:varchar(200)" json:"-"`                                             // ??????
	GuestLocale             string         `gorm:"type:varchar(20)" json:"guest_locale,omitempty"`                         // ????
	Status                  string         `gorm:"index;not null" json:"status"`                                           // ????
	Currency                string         `gorm:"not null" json:"currency"`                                               // ??
	OriginalAmount          Money          `gorm:"type:decimal(20,2);not null;default:0" json:"original_amount"`           // ????
	DiscountAmount          Money          `gorm:"type:decimal(20,2);not null;default:0" json:"discount_amount"`           // ????
	MemberDiscountAmount    Money          `gorm:"type:decimal(20,2);not null;default:0" json:"member_discount_amount"`    // ??????
	PromotionDiscountAmount Money          `gorm:"type:decimal(20,2);not null;default:0" json:"promotion_discount_amount"` // ???????
	TotalAmount             Money          `gorm:"type:decimal(20,2);not null;default:0" json:"total_amount"`              // ????
	WalletPaidAmount        Money          `gorm:"type:decimal(20,2);not null;default:0" json:"wallet_paid_amount"`        // ??????
	OnlinePaidAmount        Money          `gorm:"type:decimal(20,2);not null;default:0" json:"online_paid_amount"`        // ??????
	RefundedAmount          Money          `gorm:"type:decimal(20,2);not null;default:0" json:"refunded_amount"`           // ?????(????)
	MemberLevelID           *uint          `gorm:"index" json:"member_level_id,omitempty"`                                 // ???????
	CouponID                *uint          `gorm:"index" json:"coupon_id,omitempty"`                                       // ???ID
	PromotionID             *uint          `gorm:"index" json:"promotion_id,omitempty"`                                    // ???ID(????)
	AffiliateProfileID      *uint          `gorm:"index" json:"affiliate_profile_id,omitempty"`                            // ????????ID??
	AffiliateCode           string         `gorm:"type:varchar(32);index" json:"affiliate_code,omitempty"`                 // ??????ID??
	ClientIP                string         `gorm:"type:varchar(64)" json:"client_ip,omitempty"`                            // ?????IP
	ExpiresAt               *time.Time     `gorm:"index" json:"expires_at"`                                                // ????
	PaidAt                  *time.Time     `gorm:"index" json:"paid_at"`                                                   // ????
	CanceledAt              *time.Time     `gorm:"index" json:"canceled_at"`                                               // ????
	CreatedAt               time.Time      `gorm:"index" json:"created_at"`                                                // ????
	UpdatedAt               time.Time      `gorm:"index" json:"updated_at"`                                                // ????
	DeletedAt               gorm.DeletedAt `gorm:"index" json:"-"`                                                         // ?????

	Items []OrderItem `gorm:"foreignKey:OrderID" json:"items,omitempty"` // ???
	// ??
	Fulfillment *Fulfillment `gorm:"foreignKey:OrderID" json:"fulfillment,omitempty"` // ????
	Children    []Order      `gorm:"foreignKey:ParentID" json:"children,omitempty"`   // ???
}

// TableName ????
func (Order) TableName() string {
	return "orders"
}

// StripCostPrice ????????????,????????????
func (o *Order) StripCostPrice() {
	for i := range o.Items {
		o.Items[i].CostPrice = Money{}
	}
	for i := range o.Children {
		o.Children[i].StripCostPrice()
	}
}

// MaskUpstreamFulfillmentType ????????? upstream ??????? manual,
// ?????????????????
func (o *Order) MaskUpstreamFulfillmentType() {
	const upstream = "upstream"
	const manual = "manual"
	for i := range o.Items {
		if o.Items[i].FulfillmentType == upstream {
			o.Items[i].FulfillmentType = manual
		}
	}
	if o.Fulfillment != nil && o.Fulfillment.Type == upstream {
		o.Fulfillment.Type = manual
	}
	for i := range o.Children {
		o.Children[i].MaskUpstreamFulfillmentType()
	}
}

// TruncateFulfillmentPayload ????????????????,?????????
func (o *Order) TruncateFulfillmentPayload() {
	if o.Fulfillment != nil {
		o.Fulfillment.TruncatePayload(FulfillmentPayloadMaxPreviewLines)
	}
	for i := range o.Children {
		o.Children[i].TruncateFulfillmentPayload()
	}
}
