package models

import (
	"time"

	"gorm.io/gorm"
)

// MemberLevel ??????
type MemberLevel struct {
	ID                uint           `gorm:"primarykey" json:"id"`
	NameJSON          JSON           `gorm:"type:json;not null" json:"name"`                                  // ?????
	Slug              string         `gorm:"uniqueIndex;not null" json:"slug"`                                // ????(default/silver/gold/diamond)
	Icon              string         `gorm:"default:''" json:"icon"`                                          // ????(emoji ??? URL)
	DiscountRate      Money          `gorm:"type:decimal(6,2);not null;default:100" json:"discount_rate"`     // ?????(100=??, 90=9?, 80=8?)
	RechargeThreshold Money          `gorm:"type:decimal(20,2);not null;default:0" json:"recharge_threshold"` // ????????(0=?????)
	SpendThreshold    Money          `gorm:"type:decimal(20,2);not null;default:0" json:"spend_threshold"`    // ????????(0=?????)
	IsDefault         bool           `gorm:"not null;default:false" json:"is_default"`                        // ??????(???)
	SortOrder         int            `gorm:"not null;default:0" json:"sort_order"`                            // ????(??????)
	IsActive          bool           `gorm:"not null;default:true" json:"is_active"`                          // ????
	CreatedAt         time.Time      `gorm:"index" json:"created_at"`
	UpdatedAt         time.Time      `gorm:"index" json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

func (MemberLevel) TableName() string {
	return "member_levels"
}
