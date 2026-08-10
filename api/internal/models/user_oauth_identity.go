package models

import "time"

// UserOAuthIdentity ?????????
// ??:????????????????????
type UserOAuthIdentity struct {
	ID             uint       `gorm:"primarykey" json:"id"`                                                              // ??
	UserID         uint       `gorm:"index;not null" json:"user_id"`                                                     // ????ID
	Provider       string     `gorm:"type:varchar(32);index:idx_provider_user,unique;not null" json:"provider"`          // ???
	ProviderUserID string     `gorm:"type:varchar(128);index:idx_provider_user,unique;not null" json:"provider_user_id"` // ?????ID
	Username       string     `gorm:"type:varchar(128)" json:"username"`                                                 // ??????
	AvatarURL      string     `gorm:"type:text" json:"avatar_url"`                                                       // ????
	AuthAt         *time.Time `json:"auth_at"`                                                                           // ??????
	CreatedAt      time.Time  `gorm:"index" json:"created_at"`                                                           // ????
	UpdatedAt      time.Time  `gorm:"index" json:"updated_at"`                                                           // ????
}

// TableName ????
func (UserOAuthIdentity) TableName() string {
	return "user_oauth_identities"
}
