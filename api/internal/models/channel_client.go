package models

import (
	"time"

	"gorm.io/gorm"
)

// ChannelClient ?????(Telegram Bot ??????????)
type ChannelClient struct {
	ID            uint           `json:"id" gorm:"primaryKey"`
	Name          string         `json:"name" gorm:"size:100;not null"`
	ChannelType   string         `json:"channel_type" gorm:"size:50;not null;index"`
	ChannelKey    string         `json:"channel_key" gorm:"size:64;uniqueIndex;not null"`
	ChannelSecret string         `json:"-" gorm:"size:512;not null"`             // AES-256-GCM encrypted
	BotToken      string         `json:"-" gorm:"size:512"`                      // AES-256-GCM encrypted, Telegram Bot Token
	CallbackURL   string         `json:"callback_url" gorm:"size:500"`           // ??????(? http://localhost:8444/internal/order-fulfilled)
	Status        int            `json:"status" gorm:"default:1;not null;index"` // 1=active, 0=disabled
	Description   string         `json:"description" gorm:"size:500"`
	LastUsedAt    *time.Time     `json:"last_used_at"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}
