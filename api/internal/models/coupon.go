package models

import (
	"time"

	"gorm.io/gorm"
)

// Coupon ???
type Coupon struct {
	ID           uint           `gorm:"primarykey" json:"id"`                                      // ??
	Code         string         `gorm:"uniqueIndex;not null" json:"code"`                          // ???
	Type         string         `gorm:"not null" json:"type"`                                      // ??(fixed/percent)
	Value        Money          `gorm:"type:decimal(20,2);not null" json:"value"`                  // ??(????????)
	MinAmount    Money          `gorm:"type:decimal(20,2);not null;default:0" json:"min_amount"`   // ????
	MaxDiscount  Money          `gorm:"type:decimal(20,2);not null;default:0" json:"max_discount"` // ??????
	UsageLimit   int            `gorm:"not null;default:0" json:"usage_limit"`                     // ?????(0 ?????)
	UsedCount    int            `gorm:"not null;default:0" json:"used_count"`                      // ?????
	PerUserLimit int            `gorm:"not null;default:0" json:"per_user_limit"`                  // ??????(0 ?????)
	PaymentRoles StringArray    `gorm:"type:json" json:"payment_roles"`                            // ??????(?????)
	MemberLevels UintArray      `gorm:"type:json" json:"member_levels"`                            // ??????(?????)
	ScopeType    string         `gorm:"not null" json:"scope_type"`                                // ????(product)
	ScopeRefIDs  string         `gorm:"type:text" json:"scope_ref_ids"`                            // ????ID??(JSON??)
	StartsAt     *time.Time     `gorm:"index" json:"starts_at"`                                    // ????
	EndsAt       *time.Time     `gorm:"index" json:"ends_at"`                                      // ????
	IsActive     bool           `gorm:"not null;default:true" json:"is_active"`                    // ????
	CreatedAt    time.Time      `gorm:"index" json:"created_at"`                                   // ????
	UpdatedAt    time.Time      `gorm:"index" json:"updated_at"`                                   // ????
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`                                            // ?????
}

// TableName ????
func (Coupon) TableName() string {
	return "coupons"
}
