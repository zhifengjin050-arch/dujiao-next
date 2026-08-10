package models

import (
	"time"

	"gorm.io/gorm"
)

// OrderItem ????
type OrderItem struct {
	ID                           uint           `gorm:"primarykey" json:"id"`                                                   // ??
	OrderID                      uint           `gorm:"index;not null" json:"order_id"`                                         // ??ID
	ProductID                    uint           `gorm:"index;not null" json:"product_id"`                                       // ??ID
	SKUID                        uint           `gorm:"column:sku_id;index;not null;default:0" json:"sku_id"`                   // SKU ID
	TitleJSON                    JSON           `gorm:"type:json;not null" json:"title"`                                        // ??????
	SKUSnapshotJSON              JSON           `gorm:"type:json" json:"sku_snapshot"`                                          // SKU ??(??/??)
	Tags                         StringArray    `gorm:"type:json" json:"tags"`                                                  // ????
	UnitPrice                    Money          `gorm:"type:decimal(20,2);not null;default:0" json:"unit_price"`                // ??
	CostPrice                    Money          `gorm:"type:decimal(20,2);not null;default:0" json:"cost_price"`                // ?????
	Quantity                     int            `gorm:"not null" json:"quantity"`                                               // ??
	TotalPrice                   Money          `gorm:"type:decimal(20,2);not null;default:0" json:"total_price"`               // ??
	CouponDiscount               Money          `gorm:"type:decimal(20,2);not null;default:0" json:"coupon_discount_amount"`    // ???????
	MemberDiscount               Money          `gorm:"type:decimal(20,2);not null;default:0" json:"member_discount_amount"`    // ????????
	PromotionDiscount            Money          `gorm:"type:decimal(20,2);not null;default:0" json:"promotion_discount_amount"` // ???????
	PromotionID                  *uint          `gorm:"index" json:"promotion_id,omitempty"`                                    // ???ID
	PromotionName                string         `gorm:"-" json:"promotion_name,omitempty"`                                      // ?????
	FulfillmentType              string         `gorm:"not null" json:"fulfillment_type"`                                       // ????
	ManualFormSchemaSnapshotJSON JSON           `gorm:"type:json" json:"manual_form_schema_snapshot"`                           // ?????? schema ??
	ManualFormSubmissionJSON     JSON           `gorm:"type:json" json:"manual_form_submission"`                                // ?????????
	InstructionsJSON             JSON           `gorm:"type:json" json:"instructions"`                                          // ?????????(???)
	CreatedAt                    time.Time      `gorm:"index" json:"created_at"`                                                // ????
	UpdatedAt                    time.Time      `gorm:"index" json:"updated_at"`                                                // ????
	DeletedAt                    gorm.DeletedAt `gorm:"index" json:"-"`                                                         // ?????
}

// TableName ????
func (OrderItem) TableName() string {
	return "order_items"
}
