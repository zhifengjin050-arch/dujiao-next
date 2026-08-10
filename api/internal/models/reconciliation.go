package models

import (
	"time"
)

// ReconciliationJob ?????
type ReconciliationJob struct {
	ID              uint       `gorm:"primarykey" json:"id"`
	ConnectionID    uint       `gorm:"index;not null" json:"connection_id"`
	Type            string     `gorm:"type:varchar(20);not null" json:"type"`
	Status          string     `gorm:"type:varchar(20);not null;default:'pending'" json:"status"`
	TimeRangeStart  time.Time  `json:"time_range_start"`
	TimeRangeEnd    time.Time  `json:"time_range_end"`
	TotalCount      int        `gorm:"not null;default:0" json:"total_count"`
	MatchedCount    int        `gorm:"not null;default:0" json:"matched_count"`
	MismatchedCount int        `gorm:"not null;default:0" json:"mismatched_count"`
	ResultJSON      string     `gorm:"type:text" json:"result_json,omitempty"`
	StartedAt       *time.Time `json:"started_at,omitempty"`
	FinishedAt      *time.Time `json:"finished_at,omitempty"`
	CreatedAt       time.Time  `gorm:"index" json:"created_at"`
	UpdatedAt       time.Time  `gorm:"index" json:"updated_at"`

	Connection *SiteConnection `gorm:"foreignKey:ConnectionID" json:"connection,omitempty"`
}

// TableName ????
func (ReconciliationJob) TableName() string {
	return "reconciliation_jobs"
}

// ReconciliationItem ?????
type ReconciliationItem struct {
	ID                 uint       `gorm:"primarykey" json:"id"`
	JobID              uint       `gorm:"index;not null" json:"job_id"`
	ProcurementOrderID uint       `gorm:"index" json:"procurement_order_id"`
	LocalOrderNo       string     `gorm:"type:varchar(64)" json:"local_order_no"`
	UpstreamOrderNo    string     `gorm:"type:varchar(64)" json:"upstream_order_no"`
	LocalStatus        string     `gorm:"type:varchar(20)" json:"local_status"`
	UpstreamStatus     string     `gorm:"type:varchar(20)" json:"upstream_status"`
	LocalAmount        Money      `gorm:"type:decimal(20,2);not null;default:0" json:"local_amount"`
	UpstreamAmount     Money      `gorm:"type:decimal(20,2);not null;default:0" json:"upstream_amount"`
	MismatchType       string     `gorm:"type:varchar(40)" json:"mismatch_type,omitempty"`
	Resolved           bool       `gorm:"not null;default:false" json:"resolved"`
	ResolvedBy         *uint      `json:"resolved_by,omitempty"`
	ResolvedAt         *time.Time `json:"resolved_at,omitempty"`
	Remark             string     `gorm:"type:text" json:"remark,omitempty"`
	CreatedAt          time.Time  `gorm:"index" json:"created_at"`
}

// TableName ????
func (ReconciliationItem) TableName() string {
	return "reconciliation_items"
}
