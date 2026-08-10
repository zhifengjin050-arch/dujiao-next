package models

import (
	"time"

	"gorm.io/gorm"
)

// GiftCardBatch ?????
type GiftCardBatch struct {
	ID        uint           `gorm:"primarykey" json:"id"`                                                  // ??
	BatchNo   string         `gorm:"type:varchar(48);uniqueIndex;not null" json:"batch_no"`                 // ???
	Name      string         `gorm:"type:varchar(120);not null" json:"name"`                                // ????
	Amount    Money          `gorm:"type:decimal(20,2);not null" json:"amount"`                             // ??
	Currency  string         `gorm:"type:varchar(16);not null;default:'CNY'" json:"currency"`               // ??
	Quantity  int            `gorm:"not null;default:0" json:"quantity"`                                    // ????
	ExpiresAt *time.Time     `gorm:"index" json:"expires_at"`                                               // ????(????????)
	CreatedBy *uint          `gorm:"index" json:"created_by,omitempty"`                                     // ?????ID
	CreatedAt time.Time      `gorm:"index" json:"created_at"`                                               // ????
	UpdatedAt time.Time      `gorm:"index" json:"updated_at"`                                               // ????
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`                                                        // ?????
	Cards     []GiftCard     `gorm:"foreignKey:BatchID;constraint:OnUpdate:CASCADE" json:"cards,omitempty"` // ????
}

// TableName ????
func (GiftCardBatch) TableName() string {
	return "gift_card_batches"
}
