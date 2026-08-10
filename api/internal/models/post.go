package models

import (
	"time"

	"gorm.io/gorm"
)

// Post ??/???
type Post struct {
	ID          uint           `gorm:"primarykey" json:"id"`                    // ??
	Slug        string         `gorm:"uniqueIndex;not null" json:"slug"`        // ????
	Type        string         `gorm:"not null;index" json:"type"`              // ??(blog/notice)
	TitleJSON   JSON           `gorm:"type:json;not null" json:"title"`         // ?????
	SummaryJSON JSON           `gorm:"type:json" json:"summary"`                // ?????
	ContentJSON JSON           `gorm:"type:json" json:"content"`                // ?????
	Thumbnail   string         `json:"thumbnail"`                               // ???
	IsPublished bool           `gorm:"default:false;index" json:"is_published"` // ????
	PublishedAt *time.Time     `gorm:"index" json:"published_at"`               // ????
	CreatedAt   time.Time      `gorm:"index" json:"created_at"`                 // ????
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`                          // ?????
}

// TableName ????
func (Post) TableName() string {
	return "posts"
}
