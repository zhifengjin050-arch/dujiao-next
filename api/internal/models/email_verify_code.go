package models

import (
	"time"

	"gorm.io/gorm"
)

// EmailVerifyCode ???????
type EmailVerifyCode struct {
	ID           uint           `gorm:"primarykey" json:"id"`           // ??
	Email        string         `gorm:"index;not null" json:"email"`    // ??
	UserID       *uint          `gorm:"index" json:"user_id"`           // ????ID
	Purpose      string         `gorm:"index;not null" json:"purpose"`  // ??(register/reset)
	Code         string         `gorm:"not null" json:"-"`              // ???(??????)
	ExpiresAt    time.Time      `gorm:"index" json:"expires_at"`        // ????
	VerifiedAt   *time.Time     `gorm:"index" json:"verified_at"`       // ????
	AttemptCount int            `gorm:"default:0" json:"attempt_count"` // ????
	SentAt       time.Time      `gorm:"index" json:"sent_at"`           // ????
	CreatedAt    time.Time      `gorm:"index" json:"created_at"`        // ????
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`                 // ?????
}

// TableName ????
func (EmailVerifyCode) TableName() string {
	return "email_verify_codes"
}
