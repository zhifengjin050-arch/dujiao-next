package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// JSON ????,?????????
type JSON map[string]interface{}

// Value ?? driver.Valuer ??
func (j JSON) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan ?? sql.Scanner ??
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = make(JSON)
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		// Try to convert to string as fallback
		str := fmt.Sprintf("%v", value)
		if str == "" || str == "<nil>" {
			*j = make(JSON)
			return nil
		}
		bytes = []byte(str)
	}
	return json.Unmarshal(bytes, j)
}

// StringArray ???????,????tags?images?
type StringArray []string

// Value ?? driver.Valuer ??
func (s StringArray) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

// Scan ?? sql.Scanner ??
func (s *StringArray) Scan(value interface{}) error {
	if value == nil {
		*s = StringArray{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, s)
}

// UintArray ?????????,???? JSON
type UintArray []uint

// Value ?? driver.Valuer ??
func (u UintArray) Value() (driver.Value, error) {
	if u == nil {
		return nil, nil
	}
	return json.Marshal(u)
}

// Scan ?? sql.Scanner ??
func (u *UintArray) Scan(value interface{}) error {
	if value == nil {
		*u = UintArray{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}
	return json.Unmarshal(bytes, u)
}

// Category ???
type Category struct {
	ID        uint           `gorm:"primarykey" json:"id"`                      // ??
	ParentID  uint           `gorm:"not null;default:0;index" json:"parent_id"` // ???ID,0 ??????
	Slug      string         `gorm:"uniqueIndex;not null" json:"slug"`          // ????
	NameJSON  JSON           `gorm:"type:json;not null" json:"name"`            // ?????
	Icon      string         `gorm:"type:varchar(500)" json:"icon"`             // ????(????)
	SortOrder int            `gorm:"default:0;index" json:"sort_order"`         // ????
	CreatedAt time.Time      `gorm:"index" json:"created_at"`                   // ????
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`                            // ?????
}

// TableName ????
func (Category) TableName() string {
	return "categories"
}
