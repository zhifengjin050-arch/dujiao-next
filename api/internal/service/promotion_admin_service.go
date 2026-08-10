package service

import (
	"strings"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/repository"

	"github.com/shopspring/decimal"
)

// PromotionAdminService ???????
type PromotionAdminService struct {
	repo repository.PromotionRepository
}

// NewPromotionAdminService ?????????
func NewPromotionAdminService(repo repository.PromotionRepository) *PromotionAdminService {
	return &PromotionAdminService{repo: repo}
}

// CreatePromotionInput ???????
type CreatePromotionInput struct {
	Name       string
	Type       string
	ScopeRefID uint
	Value      models.Money
	MinAmount  models.Money
	StartsAt   *time.Time
	EndsAt     *time.Time
	IsActive   *bool
}

// UpdatePromotionInput ???????
type UpdatePromotionInput struct {
	Name       string
	Type       string
	ScopeRefID uint
	Value      models.Money
	MinAmount  models.Money
	StartsAt   *time.Time
	EndsAt     *time.Time
	IsActive   *bool
}

// Create ?????
func (s *PromotionAdminService) Create(input CreatePromotionInput) (*models.Promotion, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, ErrPromotionInvalid
	}
	if input.ScopeRefID == 0 {
		return nil, ErrPromotionInvalid
	}
	promotionType := strings.ToLower(strings.TrimSpace(input.Type))
	if promotionType != constants.PromotionTypeFixed && promotionType != constants.PromotionTypePercent && promotionType != constants.PromotionTypeSpecialPrice {
		return nil, ErrPromotionInvalid
	}
	if input.Value.Decimal.LessThanOrEqual(decimal.Zero) {
		return nil, ErrPromotionInvalid
	}
	if promotionType == constants.PromotionTypePercent && input.Value.Decimal.GreaterThan(decimal.NewFromInt(100)) {
		return nil, ErrPromotionInvalid
	}
	if input.StartsAt != nil && input.EndsAt != nil && input.EndsAt.Before(*input.StartsAt) {
		return nil, ErrPromotionInvalid
	}

	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	promotion := &models.Promotion{
		Name:       name,
		ScopeType:  constants.ScopeTypeProduct,
		ScopeRefID: input.ScopeRefID,
		Type:       promotionType,
		Value:      input.Value,
		MinAmount:  input.MinAmount,
		StartsAt:   input.StartsAt,
		EndsAt:     input.EndsAt,
		IsActive:   isActive,
	}

	if err := s.repo.Create(promotion); err != nil {
		return nil, err
	}
	return promotion, nil
}

// Update ?????
func (s *PromotionAdminService) Update(id uint, input UpdatePromotionInput) (*models.Promotion, error) {
	if id == 0 {
		return nil, ErrPromotionInvalid
	}
	existing, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrPromotionNotFound
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, ErrPromotionInvalid
	}
	if input.ScopeRefID == 0 {
		return nil, ErrPromotionInvalid
	}
	promotionType := strings.ToLower(strings.TrimSpace(input.Type))
	if promotionType != constants.PromotionTypeFixed && promotionType != constants.PromotionTypePercent && promotionType != constants.PromotionTypeSpecialPrice {
		return nil, ErrPromotionInvalid
	}
	if input.Value.Decimal.LessThanOrEqual(decimal.Zero) {
		return nil, ErrPromotionInvalid
	}
	if promotionType == constants.PromotionTypePercent && input.Value.Decimal.GreaterThan(decimal.NewFromInt(100)) {
		return nil, ErrPromotionInvalid
	}
	if input.StartsAt != nil && input.EndsAt != nil && input.EndsAt.Before(*input.StartsAt) {
		return nil, ErrPromotionInvalid
	}

	isActive := existing.IsActive
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	existing.Name = name
	existing.ScopeType = constants.ScopeTypeProduct
	existing.ScopeRefID = input.ScopeRefID
	existing.Type = promotionType
	existing.Value = input.Value
	existing.MinAmount = input.MinAmount
	existing.StartsAt = input.StartsAt
	existing.EndsAt = input.EndsAt
	existing.IsActive = isActive

	if err := s.repo.Update(existing); err != nil {
		return nil, ErrPromotionUpdateFailed
	}
	return existing, nil
}

// Delete ?????
func (s *PromotionAdminService) Delete(id uint) error {
	if id == 0 {
		return ErrPromotionInvalid
	}
	existing, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrPromotionNotFound
	}
	if err := s.repo.Delete(id); err != nil {
		return ErrPromotionDeleteFailed
	}
	return nil
}

// List ???????
func (s *PromotionAdminService) List(filter repository.PromotionListFilter) ([]models.Promotion, int64, error) {
	return s.repo.List(filter)
}
