package models

import "time"

// AffiliateClick ????????
type AffiliateClick struct {
	ID                 uint      `gorm:"primarykey" json:"id"`                                       // ??
	AffiliateProfileID uint      `gorm:"not null;index" json:"affiliate_profile_id"`                 // ????ID
	VisitorKey         string    `gorm:"type:varchar(128);index" json:"visitor_key"`                 // ????
	LandingPath        string    `gorm:"type:varchar(512)" json:"landing_path"`                      // ??????
	Referrer           string    `gorm:"type:varchar(1024)" json:"referrer"`                         // ????
	ClientIP           string    `gorm:"type:varchar(64)" json:"client_ip"`                          // ???IP
	UserAgent          string    `gorm:"type:varchar(1024)" json:"user_agent"`                       // ???UA
	CreatedAt          time.Time `gorm:"index;not null;default:CURRENT_TIMESTAMP" json:"created_at"` // ????

	AffiliateProfile AffiliateProfile `gorm:"foreignKey:AffiliateProfileID" json:"affiliate_profile,omitempty"` // ????
}

// TableName ????
func (AffiliateClick) TableName() string {
	return "affiliate_clicks"
}
