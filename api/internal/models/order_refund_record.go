package models

import (
	"time"

	"gorm.io/gorm"
)

// OrderRefundRecord ????
type OrderRefundRecord struct {
	ID         uint           `gorm:"primarykey" json:"id"`
	UserID     uint           `gorm:"index;not null;default:0" json:"user_id"`
	GuestEmail string         `gorm:"index;type:varchar(255)" json:"guest_email,omitempty"`
	OrderID    uint           `gorm:"index;not null" json:"order_id"`
	Type       string         `gorm:"index;type:varchar(32);not null" json:"type"`
	Amount     Money          `gorm:"type:decimal(20,2);not null;default:0" json:"amount"`
	Currency   string         `gorm:"type:varchar(16);not null;default:''" json:"currency"`
	Remark     string         `gorm:"type:text" json:"remark,omitempty"`
	CreatedAt  time.Time      `gorm:"index" json:"created_at"`
	UpdatedAt  time.Time      `gorm:"index" json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName ????
func (OrderRefundRecord) TableName() string {
	return "order_refund_records"
}
