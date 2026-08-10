package models

import (
	"time"

	"gorm.io/gorm"
)

// Promotion ???/????
type Promotion struct {
	ID         uint           `gorm:"primarykey" json:"id"`                                    // ??
	Name       string         `gorm:"not null" json:"name"`                                    // ??
	ScopeType  string         `gorm:"not null" json:"scope_type"`                              // ????(product)
	ScopeRefID uint           `gorm:"index;not null" json:"scope_ref_id"`                      // ????ID
	Type       string         `gorm:"not null" json:"type"`                                    // ??(fixed/percent/special_price)
	Value      Money          `gorm:"type:decimal(20,2);not null" json:"value"`                // ??(????/???/???)
	MinAmount  Money          `gorm:"type:decimal(20,2);not null;default:0" json:"min_amount"` // ????
	StartsAt   *time.Time     `gorm:"index" json:"starts_at"`                                  // ????
	EndsAt     *time.Time     `gorm:"index" json:"ends_at"`                                    // ????
	IsActive   bool           `gorm:"not null;default:true" json:"is_active"`                  // ????
	CreatedAt  time.Time      `gorm:"index" json:"created_at"`                                 // ????
	UpdatedAt  time.Time      `gorm:"index" json:"updated_at"`                                 // ????
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"-"`                                          // ?????
}

// TableName ????
func (Promotion) TableName() string {
	return "promotions"
}
