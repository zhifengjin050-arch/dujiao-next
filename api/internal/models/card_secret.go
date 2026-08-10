package models

import (
	"time"

	"gorm.io/gorm"
)

const (
	CardSecretStatusAvailable = "available"
	CardSecretStatusReserved  = "reserved"
	CardSecretStatusUsed      = "used"
)

// CardSecret ?????
type CardSecret struct {
	ID         uint           `gorm:"primarykey" json:"id"`                                                         // ??
	ProductID  uint           `gorm:"not null;index:idx_card_secret_reserve" json:"product_id"`                     // ??ID
	SKUID      uint           `gorm:"column:sku_id;not null;default:0;index:idx_card_secret_reserve" json:"sku_id"` // SKU ID
	BatchID    *uint          `gorm:"index" json:"batch_id,omitempty"`                                              // ??ID
	Secret     string         `gorm:"type:text;not null" json:"secret"`                                             // ????
	Status     string         `gorm:"not null;index:idx_card_secret_reserve" json:"status"`                         // ??(available/used)
	OrderID    *uint          `gorm:"index" json:"order_id,omitempty"`                                              // ????ID
	ReservedAt *time.Time     `gorm:"index" json:"reserved_at"`                                                     // ????
	UsedAt     *time.Time     `gorm:"index" json:"used_at"`                                                         // ????
	CreatedAt  time.Time      `gorm:"index" json:"created_at"`                                                      // ????
	UpdatedAt  time.Time      `gorm:"index" json:"updated_at"`                                                      // ????
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`                                                               // ?????

	Batch *CardSecretBatch `gorm:"foreignKey:BatchID" json:"batch,omitempty"` // ????
}

// TableName ????
func (CardSecret) TableName() string {
	return "card_secrets"
}
