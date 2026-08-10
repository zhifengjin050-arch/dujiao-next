package models

import (
	"time"

	"gorm.io/gorm"
)

// CardSecretBatch ?????
type CardSecretBatch struct {
	ID         uint           `gorm:"primarykey" json:"id"`                                 // ??
	ProductID  uint           `gorm:"index;not null" json:"product_id"`                     // ??ID
	SKUID      uint           `gorm:"column:sku_id;index;not null;default:0" json:"sku_id"` // SKU ID
	BatchNo    string         `gorm:"uniqueIndex;not null" json:"batch_no"`                 // ???
	Source     string         `gorm:"not null" json:"source"`                               // ??(manual/csv)
	TotalCount int            `gorm:"not null" json:"total_count"`                          // ???
	Note       string         `gorm:"type:text" json:"note"`                                // ??
	CreatedBy  *uint          `gorm:"index" json:"created_by,omitempty"`                    // ?????ID
	CreatedAt  time.Time      `gorm:"index" json:"created_at"`                              // ????
	UpdatedAt  time.Time      `gorm:"index" json:"updated_at"`                              // ????
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`                                       // ?????
}

// TableName ????
func (CardSecretBatch) TableName() string {
	return "card_secret_batches"
}
