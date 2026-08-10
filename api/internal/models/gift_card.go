package models

import (
	"time"

	"gorm.io/gorm"
)

const (
	GiftCardStatusActive   = "active"
	GiftCardStatusRedeemed = "redeemed"
	GiftCardStatusDisabled = "disabled"
)

// GiftCard ???
type GiftCard struct {
	ID             uint           `gorm:"primarykey" json:"id"`                                           // ??
	BatchID        *uint          `gorm:"index" json:"batch_id,omitempty"`                                // ??ID
	Name           string         `gorm:"type:varchar(120);not null" json:"name"`                         // ?????
	Code           string         `gorm:"type:varchar(80);uniqueIndex;not null" json:"code"`              // ??
	Amount         Money          `gorm:"type:decimal(20,2);not null" json:"amount"`                      // ??
	Currency       string         `gorm:"type:varchar(16);not null;default:'CNY'" json:"currency"`        // ??
	Status         string         `gorm:"type:varchar(24);index;not null;default:'active'" json:"status"` // ??
	ExpiresAt      *time.Time     `gorm:"index" json:"expires_at"`                                        // ????
	RedeemedAt     *time.Time     `gorm:"index" json:"redeemed_at"`                                       // ????
	RedeemedUserID *uint          `gorm:"index" json:"redeemed_user_id,omitempty"`                        // ????ID
	WalletTxnID    *uint          `gorm:"index" json:"wallet_txn_id,omitempty"`                           // ????ID
	CreatedAt      time.Time      `gorm:"index" json:"created_at"`                                        // ????
	UpdatedAt      time.Time      `gorm:"index" json:"updated_at"`                                        // ????
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"-"`                                                 // ?????
	Batch          *GiftCardBatch `gorm:"foreignKey:BatchID" json:"batch,omitempty"`                      // ????
}

// TableName ????
func (GiftCard) TableName() string {
	return "gift_cards"
}
