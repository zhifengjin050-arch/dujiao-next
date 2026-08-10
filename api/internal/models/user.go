package models

import (
	"time"

	"gorm.io/gorm"
)

// User ???
type User struct {
	ID                    uint           `gorm:"primarykey" json:"id"`                                         // ??
	Email                 string         `gorm:"uniqueIndex;not null" json:"email"`                            // ??
	PasswordHash          string         `gorm:"not null" json:"-"`                                            // ????(??????)
	PasswordSetupRequired bool           `gorm:"not null;default:false" json:"-"`                              // ??????????(Telegram ??????)
	DisplayName           string         `gorm:"default:''" json:"display_name"`                               // ??
	Locale                string         `gorm:"default:'zh-CN'" json:"locale"`                                // ????
	Status                string         `gorm:"default:'active'" json:"status"`                               // ????
	MemberLevelID         uint           `gorm:"not null;default:0" json:"member_level_id"`                    // ??????ID
	TotalRecharged        Money          `gorm:"type:decimal(20,2);not null;default:0" json:"total_recharged"` // ????
	TotalSpent            Money          `gorm:"type:decimal(20,2);not null;default:0" json:"total_spent"`     // ????
	AdminNote             string         `gorm:"type:text;default:''" json:"admin_note,omitempty"`             // ?????(?????)
	TokenVersion          uint64         `gorm:"not null;default:0" json:"-"`                                  // Token ??(??????)
	TokenInvalidBefore    *time.Time     `gorm:"index" json:"-"`                                               // ???????? Token ??
	EmailVerifiedAt       *time.Time     `json:"email_verified_at"`                                            // ??????
	LastLoginAt           *time.Time     `json:"last_login_at"`                                                // ??????
	CreatedAt             time.Time      `gorm:"index" json:"created_at"`                                      // ????
	UpdatedAt             time.Time      `gorm:"index" json:"updated_at"`                                      // ????
	DeletedAt             gorm.DeletedAt `gorm:"index" json:"-"`                                               // ?????
}

// TableName ????
func (User) TableName() string {
	return "users"
}
