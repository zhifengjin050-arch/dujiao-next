package repository

import (
	"errors"

	"github.com/dujiao-next/internal/models"

	"gorm.io/gorm"
)

// AdminRepository ?????????
type AdminRepository interface {
	GetByUsername(username string) (*models.Admin, error)
	GetByID(id uint) (*models.Admin, error)
	List() ([]models.Admin, error)
	Count() (int64, error)
	Create(admin *models.Admin) error
	Update(admin *models.Admin) error
	Delete(id uint) error
}

// GormAdminRepository GORM ??
type GormAdminRepository struct {
	db *gorm.DB
}

// NewAdminRepository ???????
func NewAdminRepository(db *gorm.DB) *GormAdminRepository {
	return &GormAdminRepository{db: db}
}

// GetByUsername ??????????
func (r *GormAdminRepository) GetByUsername(username string) (*models.Admin, error) {
	var admin models.Admin
	if err := r.db.Where("username = ?", username).First(&admin).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &admin, nil
}

// GetByID ?? ID ?????
func (r *GormAdminRepository) GetByID(id uint) (*models.Admin, error) {
	var admin models.Admin
	if err := r.db.First(&admin, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &admin, nil
}

// List ???????
func (r *GormAdminRepository) List() ([]models.Admin, error) {
	admins := make([]models.Admin, 0)
	err := r.db.
		Select("id", "username", "is_super", "last_login_at", "created_at").
		Order("id ASC").
		Find(&admins).Error
	if err != nil {
		return nil, err
	}
	return admins, nil
}

// Count ???????
func (r *GormAdminRepository) Count() (int64, error) {
	var count int64
	if err := r.db.Model(&models.Admin{}).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// Create ?????
func (r *GormAdminRepository) Create(admin *models.Admin) error {
	return r.db.Create(admin).Error
}

// Update ?????
func (r *GormAdminRepository) Update(admin *models.Admin) error {
	return r.db.Save(admin).Error
}

// Delete ?????(???)
func (r *GormAdminRepository) Delete(id uint) error {
	if id == 0 {
		return nil
	}
	return r.db.Delete(&models.Admin{}, id).Error
}
