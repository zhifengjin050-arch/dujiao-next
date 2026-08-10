package repository

import (
	"errors"

	"github.com/dujiao-next/internal/models"

	"gorm.io/gorm"
)

// CategoryRepository ????????
type CategoryRepository interface {
	List() ([]models.Category, error)
	GetByID(id string) (*models.Category, error)
	Create(category *models.Category) error
	Update(category *models.Category) error
	Delete(id string) error
	CountBySlug(slug string, excludeID *string) (int64, error)
	CountChildren(categoryID string) (int64, error)
	CountProducts(categoryID string) (int64, error)
	CountActiveProducts(categoryID string) (int64, error)
	GetBySlug(slug string) (*models.Category, error)
}

// GormCategoryRepository GORM ??
type GormCategoryRepository struct {
	db *gorm.DB
}

// NewCategoryRepository ??????
func NewCategoryRepository(db *gorm.DB) *GormCategoryRepository {
	return &GormCategoryRepository{db: db}
}

// List ????
func (r *GormCategoryRepository) List() ([]models.Category, error) {
	var categories []models.Category
	if err := r.db.Order("sort_order DESC, id ASC").Find(&categories).Error; err != nil {
		return nil, err
	}
	return categories, nil
}

// GetByID ?? ID ????
func (r *GormCategoryRepository) GetByID(id string) (*models.Category, error) {
	var category models.Category
	if err := r.db.First(&category, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &category, nil
}

// Create ????
func (r *GormCategoryRepository) Create(category *models.Category) error {
	return r.db.Create(category).Error
}

// Update ????
func (r *GormCategoryRepository) Update(category *models.Category) error {
	return r.db.Save(category).Error
}

// Delete ????
func (r *GormCategoryRepository) Delete(id string) error {
	return r.db.Delete(&models.Category{}, id).Error
}

// CountBySlug ?? slug ??
func (r *GormCategoryRepository) CountBySlug(slug string, excludeID *string) (int64, error) {
	var count int64
	query := r.db.Model(&models.Category{}).Where("slug = ?", slug)
	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountChildren ???????????
func (r *GormCategoryRepository) CountChildren(categoryID string) (int64, error) {
	var count int64
	if err := r.db.Model(&models.Category{}).Where("parent_id = ?", categoryID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountProducts ?????????
func (r *GormCategoryRepository) CountProducts(categoryID string) (int64, error) {
	var count int64
	if err := r.db.Model(&models.Product{}).Where("category_id = ?", categoryID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// GetBySlug ?? slug ????
func (r *GormCategoryRepository) GetBySlug(slug string) (*models.Category, error) {
	var category models.Category
	if err := r.db.Where("slug = ?", slug).First(&category).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &category, nil
}

// CountActiveProducts ????????????
func (r *GormCategoryRepository) CountActiveProducts(categoryID string) (int64, error) {
	var count int64
	if err := r.db.Model(&models.Product{}).Where("category_id = ? AND is_active = ?", categoryID, true).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
