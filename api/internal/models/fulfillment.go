package models

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

// FulfillmentPayloadMaxPreviewLines ????????(API ??)
const FulfillmentPayloadMaxPreviewLines = 100

// FulfillmentPayloadMaxEmailLines ?????????????,???????
const FulfillmentPayloadMaxEmailLines = 20

// ShouldAttachFulfillmentPayload ??????????????????
func ShouldAttachFulfillmentPayload(payload string) bool {
	if payload == "" {
		return false
	}
	return len(strings.Split(payload, "\n")) > FulfillmentPayloadMaxEmailLines
}

// Fulfillment ?????
type Fulfillment struct {
	ID               uint           `gorm:"primarykey" json:"id"`                 // ??
	OrderID          uint           `gorm:"uniqueIndex;not null" json:"order_id"` // ??ID
	Type             string         `gorm:"not null" json:"type"`                 // ????(auto/manual)
	Status           string         `gorm:"not null" json:"status"`               // ????(pending/delivered)
	Payload          string         `gorm:"type:text" json:"payload"`             // ????
	PayloadLineCount int            `gorm:"-" json:"payload_line_count"`          // ???????(????,API ?????)
	LogisticsJSON    JSON           `gorm:"type:json" json:"delivery_data"`       // ???????
	DeliveredBy      *uint          `gorm:"index" json:"delivered_by,omitempty"`  // ?????ID
	DeliveredAt      *time.Time     `gorm:"index" json:"delivered_at,omitempty"`  // ????
	CreatedAt        time.Time      `gorm:"index" json:"created_at"`              // ????
	UpdatedAt        time.Time      `gorm:"index" json:"updated_at"`              // ????
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`                       // ?????
}

// TruncatePayload ?? payload ?????? maxLines ?,?? API ???????????
func (f *Fulfillment) TruncatePayload(maxLines int) {
	if f == nil || f.Payload == "" {
		return
	}
	lines := strings.Split(f.Payload, "\n")
	f.PayloadLineCount = len(lines)
	if len(lines) > maxLines {
		f.Payload = strings.Join(lines[:maxLines], "\n")
	}
}

// TableName ????
func (Fulfillment) TableName() string {
	return "fulfillments"
}
