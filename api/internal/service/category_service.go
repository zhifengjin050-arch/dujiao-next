package service

import (
	"strconv"

	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/repository"
)

// CategoryService ??????
type CategoryService struct {
	repo repository.CategoryRepository
}

// NewCategoryService ??????
func NewCategoryService(repo repository.CategoryRepository) *CategoryService {
	return &CategoryService{repo: repo}
}

// CreateCategoryInput ??/??????
type CreateCategoryInput struct {
	ParentID  uint
	Slug      string
	NameJSON  map[string]interface{}
	Icon      string
	SortOrder int
}

// List ??????
func (s *CategoryService) List() ([]models.Category, error) {
	return s.repo.List()
}

// Create ????
func (s *CategoryService) Create(input CreateCategoryInput) (*models.Category, error) {
	if err := s.validateParent(nil, input.ParentID); err != nil {
		return nil, err
	}

	count, err := s.repo.CountBySlug(input.Slug, nil)
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, ErrSlugExists
	}

	category := models.Category{
		ParentID:  input.ParentID,
		Slug:      input.Slug,
		NameJSON:  models.JSON(input.NameJSON),
		Icon:      input.Icon,
		SortOrder: input.SortOrder,
	}
	if err := s.repo.Create(&category); err != nil {
		return nil, err
	}
	return &category, nil
}

// Update ????
func (s *CategoryService) Update(id string, input CreateCategoryInput) (*models.Category, error) {
	category, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if category == nil {
		return nil, ErrNotFound
	}
	if err := s.validateParent(category, input.ParentID); err != nil {
		return nil, err
	}

	count, err := s.repo.CountBySlug(input.Slug, &id)
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, ErrSlugExists
	}

	category.ParentID = input.ParentID
	category.Slug = input.Slug
	category.NameJSON = models.JSON(input.NameJSON)
	category.Icon = input.Icon
	category.SortOrder = input.SortOrder

	if err := s.repo.Update(category); err != nil {
		return nil, err
	}
	return category, nil
}

// Delete ????
func (s *CategoryService) Delete(id string) error {
	category, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if category == nil {
		return ErrNotFound
	}

	childCount, err := s.repo.CountChildren(id)
	if err != nil {
		return err
	}
	if childCount > 0 {
		return ErrCategoryInUse
	}

	count, err := s.repo.CountProducts(id)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrCategoryInUse
	}
	return s.repo.Delete(id)
}

func (s *CategoryService) validateParent(category *models.Category, parentID uint) error {
	if parentID == 0 {
		return nil
	}

	if category != nil && category.ID == parentID {
		return ErrCategoryParentInvalid
	}

	parentIDStr := strconv.FormatUint(uint64(parentID), 10)
	parent, err := s.repo.GetByID(parentIDStr)
	if err != nil {
		return err
	}
	if parent == nil || parent.ParentID != 0 {
		return ErrCategoryParentInvalid
	}

	if category != nil && category.ParentID == 0 {
		childCount, err := s.repo.CountChildren(strconv.FormatUint(uint64(category.ID), 10))
		if err != nil {
			return err
		}
		if childCount > 0 {
			return ErrCategoryParentInvalid
		}
	}

	return nil
}
