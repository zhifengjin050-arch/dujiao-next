package models

import (
	"time"

	"gorm.io/gorm"
)

// WalletRechargeOrder ???????
type WalletRechargeOrder struct {
	ID              uint           `gorm:"primarykey" json:"id"`                                     // ??
	RechargeNo      string         `gorm:"type:varchar(40);uniqueIndex;not null" json:"recharge_no"` // ????
	UserID          uint           `gorm:"index;not null" json:"user_id"`                            // ??ID
	PaymentID       uint           `gorm:"uniqueIndex;not null" json:"payment_id"`                   // ????ID
	ChannelID       uint           `gorm:"index;not null" json:"channel_id"`                         // ????ID
	ProviderType    string         `gorm:"type:varchar(32);not null" json:"provider_type"`           // ?????
	ChannelType     string         `gorm:"type:varchar(32);not null" json:"channel_type"`            // ????
	InteractionMode string         `gorm:"type:varchar(32);not null" json:"interaction_mode"`        // ????
	Amount          Money          `gorm:"type:decimal(20,2);not null" json:"amount"`                // ????
	PayableAmount   Money          `gorm:"type:decimal(20,2);not null" json:"payable_amount"`        // ??????(????)
	FeeRate         Money          `gorm:"type:decimal(6,2);not null;default:0" json:"fee_rate"`     // ?????
	FeeAmount       Money          `gorm:"type:decimal(20,2);not null;default:0" json:"fee_amount"`  // ?????
	Currency        string         `gorm:"type:varchar(16);not null;default:'CNY'" json:"currency"`  // ??
	Status          string         `gorm:"type:varchar(20);index;not null" json:"status"`            // ????
	Remark          string         `gorm:"type:varchar(255)" json:"remark"`                          // ??
	PaidAt          *time.Time     `gorm:"index" json:"paid_at"`                                     // ??????
	CreatedAt       time.Time      `gorm:"index" json:"created_at"`                                  // ????
	UpdatedAt       time.Time      `gorm:"index" json:"updated_at"`                                  // ????
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`                                           // ?????
}

// TableName ????
func (WalletRechargeOrder) TableName() string {
	return "wallet_recharge_orders"
}
