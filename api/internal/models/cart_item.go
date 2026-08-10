package models

import (
	"time"

	"gorm.io/gorm"
)

// CartItem ????
type CartItem struct {
	ID              uint           `gorm:"primarykey" json:"id"`                                                                 // ??
	UserID          uint           `gorm:"not null;uniqueIndex:idx_cart_user_product_sku" json:"user_id"`                        // ??ID
	ProductID       uint           `gorm:"not null;uniqueIndex:idx_cart_user_product_sku" json:"product_id"`                     // ??ID
	SKUID           uint           `gorm:"column:sku_id;not null;default:0;uniqueIndex:idx_cart_user_product_sku" json:"sku_id"` // SKU ID
	Quantity        int            `gorm:"not null" json:"quantity"`                                                             // ??
	FulfillmentType string         `gorm:"type:varchar(20);not null;default:'manual'" json:"fulfillment_type"`                   // ????
	CreatedAt       time.Time      `gorm:"index" json:"created_at"`                                                              // ????
	UpdatedAt       time.Time      `gorm:"index" json:"updated_at"`                                                              // ????
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`                                                                       // ?????

	Product *Product    `gorm:"foreignKey:ProductID" json:"product,omitempty"` // ????
	SKU     *ProductSKU `gorm:"foreignKey:SKUID" json:"sku,omitempty"`         // ??SKU
}

// TableName ????
func (CartItem) TableName() string {
	return "cart_items"
}
