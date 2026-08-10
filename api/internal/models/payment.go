package models

import (
	"time"

	"gorm.io/gorm"
)

// Payment ????
type Payment struct {
	ID              uint           `gorm:"primarykey" json:"id"`                                    // ??
	OrderID         uint           `gorm:"index;not null" json:"order_id"`                          // ??ID
	ChannelID       uint           `gorm:"index;not null" json:"channel_id"`                        // ????ID
	ProviderType    string         `gorm:"not null" json:"provider_type"`                           // ?????(official/epay)
	ChannelType     string         `gorm:"not null" json:"channel_type"`                            // ????(wechat/alipay/qqpay/paypal)
	InteractionMode string         `gorm:"not null" json:"interaction_mode"`                        // ????(qr/redirect)
	Amount          Money          `gorm:"type:decimal(20,2);not null" json:"amount"`               // ????(????)
	FeeRate         Money          `gorm:"type:decimal(6,2);not null;default:0" json:"fee_rate"`    // ?????(???)
	FixedFee        Money          `gorm:"type:decimal(6,2);not null;default:0" json:"fixed_fee"`   // ?????
	FeeAmount       Money          `gorm:"type:decimal(20,2);not null;default:0" json:"fee_amount"` // ?????
	Currency        string         `gorm:"not null" json:"currency"`                                // ??
	Status          string         `gorm:"index;not null" json:"status"`                            // ????
	ProviderRef     string         `gorm:"index" json:"provider_ref"`                               // ??????
	GatewayOrderNo  string         `gorm:"index;size:64" json:"gateway_order_no"`                   // ??????
	ProviderPayload JSON           `gorm:"type:json" json:"provider_payload"`                       // ???????
	PayURL          string         `gorm:"type:text" json:"pay_url"`                                // ????
	QRCode          string         `gorm:"type:text" json:"qr_code"`                                // ?????/??
	CreatedAt       time.Time      `gorm:"index" json:"created_at"`                                 // ????
	UpdatedAt       time.Time      `gorm:"index" json:"updated_at"`                                 // ????
	PaidAt          *time.Time     `gorm:"index" json:"paid_at"`                                    // ????
	ExpiredAt       *time.Time     `gorm:"index" json:"expired_at"`                                 // ????
	CallbackAt      *time.Time     `gorm:"index" json:"callback_at"`                                // ????
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`                                          // ?????
	ChannelName     string         `gorm:"-" json:"channel_name,omitempty"`                         // ??????(??????,JOIN ??)
}

// TableName ????
func (Payment) TableName() string {
	return "payments"
}
