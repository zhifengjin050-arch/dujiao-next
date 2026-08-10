package models

import (
	"time"

	"gorm.io/gorm"
)

// Banner ?????
type Banner struct {
	ID           uint           `gorm:"primarykey" json:"id"`                                      // ??
	Name         string         `gorm:"type:varchar(120);not null;index" json:"name"`              // ????
	Position     string         `gorm:"type:varchar(60);not null;index" json:"position"`           // ????
	TitleJSON    JSON           `gorm:"type:json" json:"title"`                                    // ?????
	SubtitleJSON JSON           `gorm:"type:json" json:"subtitle"`                                 // ??????
	Image        string         `gorm:"type:varchar(500);not null" json:"image"`                   // ??
	MobileImage  string         `gorm:"type:varchar(500)" json:"mobile_image"`                     // ?????
	LinkType     string         `gorm:"type:varchar(20);not null;default:'none'" json:"link_type"` // ????
	LinkValue    string         `gorm:"type:varchar(1000)" json:"link_value"`                      // ???
	OpenInNewTab bool           `gorm:"default:false" json:"open_in_new_tab"`                      // ???????
	IsActive     bool           `gorm:"default:true;index" json:"is_active"`                       // ????
	StartAt      *time.Time     `gorm:"index" json:"start_at"`                                     // ????
	EndAt        *time.Time     `gorm:"index" json:"end_at"`                                       // ????
	SortOrder    int            `gorm:"default:0;index" json:"sort_order"`                         // ??
	CreatedAt    time.Time      `gorm:"index" json:"created_at"`                                   // ????
	UpdatedAt    time.Time      `json:"updated_at"`                                                // ????
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`                                            // ???
}

// TableName ????
func (Banner) TableName() string {
	return "banners"
}
