package repository

import (
	"errors"
	"strings"

	"github.com/dujiao-next/internal/models"

	"gorm.io/gorm"
)

// MediaRepository ????????
type MediaRepository interface {
	List(filter MediaListFilter) ([]models.Media, int64, error)
	GetByID(id uint) (*models.Media, error)
	GetByPath(path string) (*models.Media, error)
	Create(media *models.Media) error
	Update(media *models.Media) error
	Delete(id uint) error
}

// GormMediaRepository GORM ??
type GormMediaRepository struct {
	db *gorm.DB
}

// NewMediaRepository ??????
func NewMediaRepository(db *gorm.DB) *GormMediaRepository {
	return &GormMediaRepository{db: db}
}

// List ????
func (r *GormMediaRepository) List(filter MediaListFilter) ([]models.Media, int64, error) {
	var items []models.Media
	query := r.db.Model(&models.Media{})

	if filter.Scene != "" {
		query = query.Where("scene = ?", filter.Scene)
	}
	if search := strings.TrimSpace(filter.Search); search != "" {
		like := "%" + search + "%"
		likeOp := determineLikeOp(r.db)
		query = query.Where("name "+likeOp+" ? OR filename "+likeOp+" ?", like, like)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query = applyPagination(query, filter.Page, filter.PageSize)

	if err := query.Order("created_at DESC").Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// GetByID ?? ID ????
func (r *GormMediaRepository) GetByID(id uint) (*models.Media, error) {
	var media models.Media
	if err := r.db.First(&media, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &media, nil
}

// GetByPath ????????
func (r *GormMediaRepository) GetByPath(path string) (*models.Media, error) {
	var media models.Media
	if err := r.db.Where("path = ?", path).First(&media).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &media, nil
}

// Create ??????
func (r *GormMediaRepository) Create(media *models.Media) error {
	return r.db.Create(media).Error
}

// Update ??????
func (r *GormMediaRepository) Update(media *models.Media) error {
	return r.db.Save(media).Error
}

// Delete ???????
func (r *GormMediaRepository) Delete(id uint) error {
	return r.db.Delete(&models.Media{}, id).Error
}

// determineLikeOp ????????? LIKE ???
func determineLikeOp(db *gorm.DB) string {
	if db.Dialector.Name() == "postgres" {
		return "ILIKE"
	}
	return "LIKE"
}
