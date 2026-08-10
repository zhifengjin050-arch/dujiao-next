package models

import (
	"time"

	"gorm.io/gorm"
)

// Media ???
type Media struct {
	ID        uint           `gorm:"primarykey" json:"id"`
	Name      string         `gorm:"type:varchar(255);not null;index" json:"name"`       // ???????(??=?????,??????)
	Filename  string         `gorm:"type:varchar(255);not null" json:"filename"`         // ?????(???)
	Path      string         `gorm:"type:varchar(500);not null;uniqueIndex" json:"path"` // /uploads/scene/year/month/uuid.ext
	MimeType  string         `gorm:"type:varchar(100);not null" json:"mime_type"`
	Size      int64          `gorm:"not null" json:"size"`
	Scene     string         `gorm:"type:varchar(60);not null;index" json:"scene"`
	Width     int            `gorm:"not null;default:0" json:"width"`
	Height    int            `gorm:"not null;default:0" json:"height"`
	CreatedAt time.Time      `gorm:"index" json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Media) TableName() string {
	return "media"
}
