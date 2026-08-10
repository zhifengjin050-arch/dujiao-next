package models

import "time"

// UserLoginLog ??????
// ??:?????????????,????????????????
type UserLoginLog struct {
	ID          uint      `gorm:"primarykey" json:"id"`                       // ??
	UserID      uint      `gorm:"index" json:"user_id"`                       // ??ID(?????0)
	Email       string    `gorm:"index;not null" json:"email"`                // ??????
	Status      string    `gorm:"index;not null" json:"status"`               // ????(success/failed)
	FailReason  string    `gorm:"index" json:"fail_reason"`                   // ??????
	ClientIP    string    `gorm:"type:varchar(64);index" json:"client_ip"`    // ???IP
	UserAgent   string    `gorm:"type:text" json:"user_agent"`                // ???UA
	LoginSource string    `gorm:"type:varchar(32);index" json:"login_source"` // ????(web)
	RequestID   string    `gorm:"type:varchar(64);index" json:"request_id"`   // ????ID
	CreatedAt   time.Time `gorm:"index" json:"created_at"`                    // ????
}

// TableName ????
func (UserLoginLog) TableName() string {
	return "user_login_logs"
}
