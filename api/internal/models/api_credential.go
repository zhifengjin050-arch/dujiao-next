package models

import (
	"time"

	"gorm.io/gorm"
)

// ApiCredential API ???(???? + admin ??)
type ApiCredential struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	UserID       uint           `gorm:"uniqueIndex;not null" json:"user_id"`
	ApiKey       string         `gorm:"type:varchar(64);uniqueIndex" json:"api_key"`
	ApiSecret    string         `gorm:"type:varchar(256)" json:"-"`
	Status       string         `gorm:"type:varchar(20);not null;default:'pending_review'" json:"status"`
	RejectReason string         `gorm:"type:varchar(500)" json:"reject_reason,omitempty"`
	ApprovedAt   *time.Time     `json:"approved_at,omitempty"`
	IsActive     bool           `gorm:"not null;default:false" json:"is_active"`
	LastUsedAt   *time.Time     `json:"last_used_at,omitempty"`
	CreatedAt    time.Time      `gorm:"index" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"index" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`

	// ??
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName ????
func (ApiCredential) TableName() string {
	return "api_credentials"
}
