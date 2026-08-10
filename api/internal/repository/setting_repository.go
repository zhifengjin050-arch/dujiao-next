package repository

import (
	"errors"

	"github.com/dujiao-next/internal/models"

	"gorm.io/gorm"
)

// SettingRepository ????????
type SettingRepository interface {
	GetByKey(key string) (*models.Setting, error)
	Upsert(key string, value models.JSON) (*models.Setting, error)
}

// GormSettingRepository GORM ??
type GormSettingRepository struct {
	db *gorm.DB
}

// NewSettingRepository ??????
func NewSettingRepository(db *gorm.DB) *GormSettingRepository {
	return &GormSettingRepository{db: db}
}

// GetByKey ????
func (r *GormSettingRepository) GetByKey(key string) (*models.Setting, error) {
	var setting models.Setting
	if err := r.db.Where("key = ?", key).First(&setting).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &setting, nil
}

// Upsert ???????
func (r *GormSettingRepository) Upsert(key string, value models.JSON) (*models.Setting, error) {
	setting, err := r.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if setting == nil {
		setting = &models.Setting{
			Key:       key,
			ValueJSON: value,
		}
		if err := r.db.Create(setting).Error; err != nil {
			return nil, err
		}
		return setting, nil
	}

	setting.ValueJSON = value
	if err := r.db.Save(setting).Error; err != nil {
		return nil, err
	}
	return setting, nil
}
