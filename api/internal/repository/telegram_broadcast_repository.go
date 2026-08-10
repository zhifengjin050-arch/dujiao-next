package repository

import (
	"errors"

	"github.com/dujiao-next/internal/models"

	"gorm.io/gorm"
)

// TelegramBroadcastListFilter Telegram ???????
type TelegramBroadcastListFilter struct {
	Page     int
	PageSize int
}

// TelegramBroadcastRepository Telegram ???????
type TelegramBroadcastRepository interface {
	Create(broadcast *models.TelegramBroadcast) error
	GetByID(id uint) (*models.TelegramBroadcast, error)
	List(filter TelegramBroadcastListFilter) ([]models.TelegramBroadcast, int64, error)
	Update(broadcast *models.TelegramBroadcast) error
}

// GormTelegramBroadcastRepository GORM ???
type GormTelegramBroadcastRepository struct {
	db *gorm.DB
}

// NewTelegramBroadcastRepository ?? Telegram ?????
func NewTelegramBroadcastRepository(db *gorm.DB) *GormTelegramBroadcastRepository {
	return &GormTelegramBroadcastRepository{db: db}
}

// Create ???????
func (r *GormTelegramBroadcastRepository) Create(broadcast *models.TelegramBroadcast) error {
	if broadcast == nil {
		return nil
	}
	return r.db.Create(broadcast).Error
}

// GetByID ? ID ???????
func (r *GormTelegramBroadcastRepository) GetByID(id uint) (*models.TelegramBroadcast, error) {
	if id == 0 {
		return nil, nil
	}
	var broadcast models.TelegramBroadcast
	if err := r.db.First(&broadcast, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &broadcast, nil
}

// List ?????????
func (r *GormTelegramBroadcastRepository) List(filter TelegramBroadcastListFilter) ([]models.TelegramBroadcast, int64, error) {
	query := r.db.Model(&models.TelegramBroadcast{})

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if filter.PageSize > 0 {
		query = applyPagination(query, filter.Page, filter.PageSize)
	}

	var items []models.TelegramBroadcast
	if err := query.Order("created_at DESC").Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// Update ???????
func (r *GormTelegramBroadcastRepository) Update(broadcast *models.TelegramBroadcast) error {
	if broadcast == nil {
		return nil
	}
	return r.db.Save(broadcast).Error
}
