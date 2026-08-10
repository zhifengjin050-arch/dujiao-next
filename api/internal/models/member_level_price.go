package models

import (
	"time"

	"gorm.io/gorm"
)

// MemberLevelPrice ??/SKU ??????
type MemberLevelPrice struct {
	ID            uint           `gorm:"primarykey" json:"id"`
	MemberLevelID uint           `gorm:"uniqueIndex:idx_member_level_price;not null" json:"member_level_id"`                // ????
	ProductID     uint           `gorm:"uniqueIndex:idx_member_level_price;not null" json:"product_id"`                     // ????
	SKUID         uint           `gorm:"column:sku_id;uniqueIndex:idx_member_level_price;not null;default:0" json:"sku_id"` // 0=?????,>0=SKU???
	PriceAmount   Money          `gorm:"type:decimal(20,2);not null;default:0" json:"price_amount"`                         // ????
	CreatedAt     time.Time      `gorm:"index" json:"created_at"`
	UpdatedAt     time.Time      `gorm:"index" json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

func (MemberLevelPrice) TableName() string {
	return "member_level_prices"
}
