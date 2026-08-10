package models

import (
	"time"

	"gorm.io/gorm"
)

// TelegramBroadcast Telegram Bot ?????
type TelegramBroadcast struct {
	ID               uint           `gorm:"primarykey" json:"id"`
	Title            string         `gorm:"size:200;not null" json:"title"`
	RecipientType    string         `gorm:"size:32;not null;index" json:"recipient_type"`
	FiltersJSON      JSON           `gorm:"type:json" json:"filters"`
	RecipientChatIDs StringArray    `gorm:"type:json" json:"-"`
	RecipientCount   int            `gorm:"not null;default:0" json:"recipient_count"`
	SuccessCount     int            `gorm:"not null;default:0" json:"success_count"`
	FailedCount      int            `gorm:"not null;default:0" json:"failed_count"`
	Status           string         `gorm:"size:32;not null;default:'pending';index" json:"status"`
	MessageHTML      string         `gorm:"type:text;not null" json:"message_html"`
	AttachmentURL    string         `gorm:"size:500" json:"attachment_url"`
	AttachmentName   string         `gorm:"size:255" json:"attachment_name"`
	StartedAt        *time.Time     `json:"started_at"`
	CompletedAt      *time.Time     `json:"completed_at"`
	LastError        string         `gorm:"type:text" json:"last_error"`
	CreatedAt        time.Time      `gorm:"index" json:"created_at"`
	UpdatedAt        time.Time      `gorm:"index" json:"updated_at"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName ?????
func (TelegramBroadcast) TableName() string {
	return "telegram_broadcasts"
}
