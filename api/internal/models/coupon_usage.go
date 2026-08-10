package models

import (
	"time"

	"gorm.io/gorm"
)

// CouponUsage ???????
type CouponUsage struct {
	ID             uint           `gorm:"primarykey" json:"id"`                                         // ??
	CouponID       uint           `gorm:"index;not null" json:"coupon_id"`                              // ???ID
	UserID         uint           `gorm:"index;not null" json:"user_id"`                                // ??ID
	OrderID        uint           `gorm:"index;not null" json:"order_id"`                               // ??ID
	DiscountAmount Money          `gorm:"type:decimal(20,2);not null;default:0" json:"discount_amount"` // ????
	CreatedAt      time.Time      `gorm:"index" json:"created_at"`                                      // ????
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`                                               // ?????
}

// TableName ????
func (CouponUsage) TableName() string {
	return "coupon_usages"
}
