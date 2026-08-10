package models

import (
	"time"

	"gorm.io/gorm"
)

// Admin ????
type Admin struct {
	ID                 uint           `gorm:"primarykey" json:"id"`                         // ??
	Username           string         `gorm:"uniqueIndex;not null" json:"username"`         // ?????
	PasswordHash       string         `gorm:"not null" json:"-"`                            // ????(??????)
	TokenVersion       uint64         `gorm:"not null;default:0" json:"-"`                  // Token ??(??????)
	TokenInvalidBefore *time.Time     `gorm:"index" json:"-"`                               // ???????? Token ??
	IsSuper            bool           `gorm:"not null;default:false;index" json:"is_super"` // ???????(?????)
	LastLoginAt        *time.Time     `json:"last_login_at"`                                // ??????
	CreatedAt          time.Time      `gorm:"index" json:"created_at"`                      // ????
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`                               // ?????
}

// TableName ????
func (Admin) TableName() string {
	return "admins"
}
