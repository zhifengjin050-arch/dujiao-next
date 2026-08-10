package models

import (
	"time"

	"gorm.io/gorm"
)

// AffiliateCommission ????????
type AffiliateCommission struct {
	ID                 uint           `gorm:"primarykey" json:"id"`                                                                                          // ??
	AffiliateProfileID uint           `gorm:"not null;index;index:idx_affiliate_commission_unique,unique" json:"affiliate_profile_id"`                       // ????ID
	OrderID            uint           `gorm:"not null;index;index:idx_affiliate_commission_unique,unique" json:"order_id"`                                   // ??ID
	OrderItemID        *uint          `gorm:"index" json:"order_item_id,omitempty"`                                                                          // ???ID
	CommissionType     string         `gorm:"type:varchar(20);not null;default:'order';index:idx_affiliate_commission_unique,unique" json:"commission_type"` // ????
	BaseAmount         Money          `gorm:"type:decimal(20,2);not null;default:0" json:"base_amount"`                                                      // ??????
	RatePercent        Money          `gorm:"type:decimal(10,2);not null;default:0" json:"rate_percent"`                                                     // ????(???)
	CommissionAmount   Money          `gorm:"type:decimal(20,2);not null;default:0" json:"commission_amount"`                                                // ????
	Status             string         `gorm:"type:varchar(32);not null;index" json:"status"`                                                                 // ????
	ConfirmAt          *time.Time     `gorm:"index" json:"confirm_at,omitempty"`                                                                             // ???????
	AvailableAt        *time.Time     `gorm:"index" json:"available_at,omitempty"`                                                                           // ??????
	WithdrawRequestID  *uint          `gorm:"index" json:"withdraw_request_id,omitempty"`                                                                    // ??????
	InvalidReason      string         `gorm:"type:varchar(255)" json:"invalid_reason"`                                                                       // ????
	CreatedAt          time.Time      `gorm:"index" json:"created_at"`                                                                                       // ????
	UpdatedAt          time.Time      `gorm:"index" json:"updated_at"`                                                                                       // ????
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`                                                                                                // ?????

	AffiliateProfile AffiliateProfile          `gorm:"foreignKey:AffiliateProfileID" json:"affiliate_profile,omitempty"` // ????
	Order            Order                     `gorm:"foreignKey:OrderID" json:"order,omitempty"`                        // ????
	WithdrawRequest  *AffiliateWithdrawRequest `gorm:"foreignKey:WithdrawRequestID" json:"withdraw_request,omitempty"`   // ????
}

// TableName ????
func (AffiliateCommission) TableName() string {
	return "affiliate_commissions"
}
