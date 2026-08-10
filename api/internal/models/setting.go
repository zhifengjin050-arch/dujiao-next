package models

// Setting ?????(?????)
type Setting struct {
	Key       string `gorm:"primarykey" json:"key"`  // ???
	ValueJSON JSON   `gorm:"type:json" json:"value"` // ???
}

// TableName ????
func (Setting) TableName() string {
	return "settings"
}
