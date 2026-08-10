package models

import "time"

// AuthzAuditLog ????????
// ??:???????????????,??????????????
type AuthzAuditLog struct {
	ID               uint      `gorm:"primarykey" json:"id"`
	OperatorAdminID  uint      `gorm:"index;not null" json:"operator_admin_id"`
	OperatorUsername string    `gorm:"type:varchar(100);index;not null;default:''" json:"operator_username"`
	TargetAdminID    *uint     `gorm:"index" json:"target_admin_id,omitempty"`
	TargetUsername   string    `gorm:"type:varchar(100);index;not null;default:''" json:"target_username"`
	Action           string    `gorm:"type:varchar(100);index;not null" json:"action"`
	Role             string    `gorm:"type:varchar(120);index;not null;default:''" json:"role"`
	Object           string    `gorm:"type:varchar(255);index;not null;default:''" json:"object"`
	Method           string    `gorm:"type:varchar(20);index;not null;default:''" json:"method"`
	RequestID        string    `gorm:"type:varchar(64);index;not null;default:''" json:"request_id"`
	DetailJSON       JSON      `gorm:"type:json" json:"detail"`
	CreatedAt        time.Time `gorm:"index" json:"created_at"`
}

// TableName ????
func (AuthzAuditLog) TableName() string {
	return "authz_audit_logs"
}
