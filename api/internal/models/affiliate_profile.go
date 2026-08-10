package models

import (
	"time"

	"gorm.io/gorm"
)

// AffiliateProfile ????????
type AffiliateProfile struct {
	ID            uint           `gorm:"primarykey" json:"id"`                              // ??
	UserID        uint           `gorm:"not null;uniqueIndex" json:"user_id"`               // ??ID
	AffiliateCode string         `gorm:"type:varchar(32);not null;uniqueIndex" json:"code"` // ???ID
	Status        string         `gorm:"type:varchar(20);not null;index" json:"status"`     // ??
	CreatedAt     time.Time      `gorm:"index" json:"created_at"`                           // ????
	UpdatedAt     time.Time      `gorm:"index" json:"updated_at"`                           // ????
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`                                    // ?????

	User User `gorm:"foreignKey:UserID" json:"user,omitempty"` // ????
}

// TableName ????
func (AffiliateProfile) TableName() string {
	return "affiliate_profiles"
}
