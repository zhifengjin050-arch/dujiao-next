package repository

import (
	"errors"
	"strings"

	"github.com/dujiao-next/internal/models"

	"gorm.io/gorm"
)

// PostRepository ????????
type PostRepository interface {
	List(filter PostListFilter) ([]models.Post, int64, error)
	GetBySlug(slug string, onlyPublished bool) (*models.Post, error)
	GetByID(id string) (*models.Post, error)
	Create(post *models.Post) error
	Update(post *models.Post) error
	Delete(id string) error
	CountBySlug(slug string, excludeID *string) (int64, error)
}

// GormPostRepository GORM ??
type GormPostRepository struct {
	db *gorm.DB
}

// NewPostRepository ??????
func NewPostRepository(db *gorm.DB) *GormPostRepository {
	return &GormPostRepository{db: db}
}

// List ????
func (r *GormPostRepository) List(filter PostListFilter) ([]models.Post, int64, error) {
	var posts []models.Post
	query := r.db.Model(&models.Post{})

	if filter.OnlyPublished {
		query = query.Where("is_published = ?", true)
	}
	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}
	if search := strings.TrimSpace(filter.Search); search != "" {
		like := "%" + search + "%"
		condition, argCount := buildLocalizedLikeCondition(r.db, []string{"slug"}, []string{"title_json"})
		query = query.Where(condition, repeatLikeArgs(like, argCount)...)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query = applyPagination(query, filter.Page, filter.PageSize)

	orderBy := filter.OrderBy
	if orderBy == "" {
		orderBy = "created_at DESC"
	}

	if err := query.Order(orderBy).Find(&posts).Error; err != nil {
		return nil, 0, err
	}
	return posts, total, nil
}

// GetBySlug ?? slug ????
func (r *GormPostRepository) GetBySlug(slug string, onlyPublished bool) (*models.Post, error) {
	query := r.db.Where("slug = ?", slug)
	if onlyPublished {
		query = query.Where("is_published = ?", true)
	}

	var post models.Post
	if err := query.First(&post).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &post, nil
}

// GetByID ?? ID ????
func (r *GormPostRepository) GetByID(id string) (*models.Post, error) {
	var post models.Post
	if err := r.db.First(&post, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &post, nil
}

// Create ????
func (r *GormPostRepository) Create(post *models.Post) error {
	return r.db.Create(post).Error
}

// Update ????
func (r *GormPostRepository) Update(post *models.Post) error {
	return r.db.Save(post).Error
}

// Delete ????
func (r *GormPostRepository) Delete(id string) error {
	return r.db.Delete(&models.Post{}, id).Error
}

// CountBySlug ?? slug ??
func (r *GormPostRepository) CountBySlug(slug string, excludeID *string) (int64, error) {
	var count int64
	query := r.db.Model(&models.Post{}).Where("slug = ?", slug)
	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}
	if err := query.Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
