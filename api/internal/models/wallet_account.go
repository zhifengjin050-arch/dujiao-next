package models

import (
	"time"

	"gorm.io/gorm"
)

// WalletAccount ??????
type WalletAccount struct {
	ID        uint           `gorm:"primarykey" json:"id"`                                 // ??
	UserID    uint           `gorm:"uniqueIndex;not null" json:"user_id"`                  // ??ID
	Balance   Money          `gorm:"type:decimal(20,2);not null;default:0" json:"balance"` // ????
	CreatedAt time.Time      `gorm:"index" json:"created_at"`                              // ????
	UpdatedAt time.Time      `gorm:"index" json:"updated_at"`                              // ????
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`                                       // ?????
}

// TableName ????
func (WalletAccount) TableName() string {
	return "wallet_accounts"
}
