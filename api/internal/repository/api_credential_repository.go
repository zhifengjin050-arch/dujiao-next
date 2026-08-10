package repository

import (
	"errors"

	"github.com/dujiao-next/internal/models"

	"gorm.io/gorm"
)

// ApiCredentialRepository API ????????
type ApiCredentialRepository interface {
	GetByID(id uint) (*models.ApiCredential, error)
	GetByUserID(userID uint) (*models.ApiCredential, error)
	GetAnyByUserID(userID uint) (*models.ApiCredential, error)
	GetByApiKey(apiKey string) (*models.ApiCredential, error)
	Create(cred *models.ApiCredential) error
	Update(cred *models.ApiCredential) error
	UpdateAny(cred *models.ApiCredential) error
	Delete(id uint) error
	List(filter ApiCredentialListFilter) ([]models.ApiCredential, int64, error)
}

// ApiCredentialListFilter ??????
type ApiCredentialListFilter struct {
	Status string
	UserID uint
	Search string // ????????
	Pagination
}

// GormApiCredentialRepository GORM ??
type GormApiCredentialRepository struct {
	db *gorm.DB
}

// NewApiCredentialRepository ??????
func NewApiCredentialRepository(db *gorm.DB) *GormApiCredentialRepository {
	return &GormApiCredentialRepository{db: db}
}

// GetByID ?? ID ??
func (r *GormApiCredentialRepository) GetByID(id uint) (*models.ApiCredential, error) {
	var cred models.ApiCredential
	if err := r.db.First(&cred, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &cred, nil
}

// GetByUserID ???? ID ??
func (r *GormApiCredentialRepository) GetByUserID(userID uint) (*models.ApiCredential, error) {
	var cred models.ApiCredential
	if err := r.db.Where("user_id = ?", userID).First(&cred).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &cred, nil
}

// GetAnyByUserID ???? ID ??,????????
func (r *GormApiCredentialRepository) GetAnyByUserID(userID uint) (*models.ApiCredential, error) {
	var cred models.ApiCredential
	if err := r.db.Unscoped().Where("user_id = ?", userID).First(&cred).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &cred, nil
}

// GetByApiKey ?? API Key ??(??? User ??????)
func (r *GormApiCredentialRepository) GetByApiKey(apiKey string) (*models.ApiCredential, error) {
	var cred models.ApiCredential
	if err := r.db.Preload("User").Where("api_key = ?", apiKey).First(&cred).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &cred, nil
}

// Create ????
func (r *GormApiCredentialRepository) Create(cred *models.ApiCredential) error {
	return r.db.Create(cred).Error
}

// Update ????
func (r *GormApiCredentialRepository) Update(cred *models.ApiCredential) error {
	return r.db.Save(cred).Error
}

// UpdateAny ????,????????
func (r *GormApiCredentialRepository) UpdateAny(cred *models.ApiCredential) error {
	return r.db.Unscoped().Save(cred).Error
}

// Delete ?????
func (r *GormApiCredentialRepository) Delete(id uint) error {
	return r.db.Delete(&models.ApiCredential{}, id).Error
}

// List ????
func (r *GormApiCredentialRepository) List(filter ApiCredentialListFilter) ([]models.ApiCredential, int64, error) {
	var creds []models.ApiCredential
	var total int64

	q := r.db.Model(&models.ApiCredential{})
	if filter.Status != "" {
		q = q.Where("status = ?", filter.Status)
	}
	if filter.UserID > 0 {
		q = q.Where("user_id = ?", filter.UserID)
	}
	if filter.Search != "" {
		// ????????:?? JOIN users ?
		q = q.Joins("JOIN users ON users.id = api_credentials.user_id").
			Where("users.email LIKE ? OR users.display_name LIKE ?",
				"%"+filter.Search+"%", "%"+filter.Search+"%")
	}

	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	q = q.Order("api_credentials.created_at DESC")
	if filter.Page > 0 && filter.PageSize > 0 {
		q = q.Offset((filter.Page - 1) * filter.PageSize).Limit(filter.PageSize)
	}

	if err := q.Preload("User").Find(&creds).Error; err != nil {
		return nil, 0, err
	}

	return creds, total, nil
}
