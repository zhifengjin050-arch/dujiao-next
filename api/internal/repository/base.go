package repository

import "gorm.io/gorm"

// BaseRepository ?????????????
type BaseRepository struct {
	db *gorm.DB
}

// Transaction ????????
func (b *BaseRepository) Transaction(fn func(tx *gorm.DB) error) error {
	if fn == nil {
		return nil
	}
	return b.db.Transaction(fn)
}
