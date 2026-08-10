package models

import (
	"time"

	"gorm.io/gorm"
)

// WalletTransaction ??????
type WalletTransaction struct {
	ID            uint           `gorm:"primarykey" json:"id"`                                        // ??
	UserID        uint           `gorm:"index;not null" json:"user_id"`                               // ??ID
	OrderID       *uint          `gorm:"index" json:"order_id,omitempty"`                             // ????ID
	Type          string         `gorm:"type:varchar(40);index;not null" json:"type"`                 // ????
	Direction     string         `gorm:"type:varchar(16);index;not null" json:"direction"`            // ????
	Amount        Money          `gorm:"type:decimal(20,2);not null" json:"amount"`                   // ????
	BalanceBefore Money          `gorm:"type:decimal(20,2);not null;default:0" json:"balance_before"` // ?????
	BalanceAfter  Money          `gorm:"type:decimal(20,2);not null;default:0" json:"balance_after"`  // ?????
	Currency      string         `gorm:"type:varchar(16);not null;default:'CNY'" json:"currency"`     // ??
	Reference     string         `gorm:"type:varchar(120);uniqueIndex" json:"reference"`              // ?????
	Remark        string         `gorm:"type:varchar(255)" json:"remark"`                             // ??
	CreatedAt     time.Time      `gorm:"index" json:"created_at"`                                     // ????
	UpdatedAt     time.Time      `gorm:"index" json:"updated_at"`                                     // ????
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`                                              // ?????
}

// TableName ????
func (WalletTransaction) TableName() string {
	return "wallet_transactions"
}
