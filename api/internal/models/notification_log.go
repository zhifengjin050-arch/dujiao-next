package models

import "time"

// NotificationLog ??????
// ??:?????????????????,??????????????????????
type NotificationLog struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	EventType     string    `gorm:"type:varchar(100);index;not null;default:''" json:"event_type"`
	BizType       string    `gorm:"type:varchar(100);index;not null;default:''" json:"biz_type"`
	BizID         uint      `gorm:"index;not null;default:0" json:"biz_id"`
	Channel       string    `gorm:"type:varchar(32);index;not null;default:''" json:"channel"`
	Recipient     string    `gorm:"type:varchar(255);index;not null;default:''" json:"recipient"`
	Locale        string    `gorm:"type:varchar(20);index;not null;default:''" json:"locale"`
	Title         string    `gorm:"type:text;not null" json:"title"`
	Body          string    `gorm:"type:text;not null" json:"body"`
	Status        string    `gorm:"type:varchar(32);index;not null;default:''" json:"status"`
	ErrorMessage  string    `gorm:"type:text" json:"error_message"`
	IsTest        bool      `gorm:"index;not null;default:false" json:"is_test"`
	VariablesJSON JSON      `gorm:"type:json" json:"variables"`
	CreatedAt     time.Time `gorm:"index" json:"created_at"`
}

// TableName ????
func (NotificationLog) TableName() string {
	return "notification_logs"
}
