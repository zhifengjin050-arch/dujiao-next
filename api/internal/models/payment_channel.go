package models

import (
	"time"

	"gorm.io/gorm"
)

// PaymentChannel ??????
type PaymentChannel struct {
	ID                 uint           `gorm:"primarykey" json:"id"`                                    // ??
	Name               string         `gorm:"not null" json:"name"`                                    // ????
	Icon               string         `gorm:"type:varchar(512);default:''" json:"icon"`                // ????(??)
	ProviderType       string         `gorm:"not null" json:"provider_type"`                           // ?????(official/epay)
	ChannelType        string         `gorm:"not null" json:"channel_type"`                            // ????(wechat/alipay/qqpay/paypal)
	InteractionMode    string         `gorm:"not null" json:"interaction_mode"`                        // ????(qr/redirect)
	FeeRate            Money          `gorm:"type:decimal(6,2);not null;default:0" json:"fee_rate"`    // ?????(???)
	FixedFee           Money          `gorm:"type:decimal(6,2);not null;default:0" json:"fixed_fee"`   // ?????
	MinAmount          Money          `gorm:"type:decimal(20,2);not null;default:0" json:"min_amount"` // ??????(0=??)
	MaxAmount          Money          `gorm:"type:decimal(20,2);not null;default:0" json:"max_amount"` // ??????(0=??)
	HideAmountOutRange bool           `gorm:"not null;default:false" json:"hide_amount_out_range"`     // ?????????
	PaymentRoles       StringArray    `gorm:"type:json" json:"payment_roles"`                          // ??????
	MemberLevels       UintArray      `gorm:"type:json" json:"member_levels"`                          // ??????
	PaymentTypes       StringArray    `gorm:"type:json" json:"payment_types"`                          // ??????
	ConfigJSON         JSON           `gorm:"type:json" json:"config_json"`                            // ????
	IsActive           bool           `gorm:"index;not null;default:true" json:"is_active"`            // ????
	SortOrder          int            `gorm:"not null;default:0" json:"sort_order"`                    // ??
	CreatedAt          time.Time      `gorm:"index" json:"created_at"`                                 // ????
	UpdatedAt          time.Time      `gorm:"index" json:"updated_at"`                                 // ????
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`                                          // ?????
}

// TableName ????
func (PaymentChannel) TableName() string {
	return "payment_channels"
}
