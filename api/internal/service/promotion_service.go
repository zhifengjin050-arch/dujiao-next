package service

import (
	"strings"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/repository"

	"github.com/shopspring/decimal"
)

// PromotionService ?????
type PromotionService struct {
	promotionRepo repository.PromotionRepository
}

// NewPromotionService ???????
func NewPromotionService(promotionRepo repository.PromotionRepository) *PromotionService {
	return &PromotionService{
		promotionRepo: promotionRepo,
	}
}

// GetProductPromotions ????????????(??????)
func (s *PromotionService) GetProductPromotions(productID uint) ([]models.Promotion, error) {
	return s.promotionRepo.GetAllActiveByProduct(productID, time.Now())
}

// ApplyPromotion ???????(??????)
func (s *PromotionService) ApplyPromotion(product *models.Product, quantity int) (*models.Promotion, models.Money, error) {
	if product == nil || quantity <= 0 {
		return nil, models.Money{}, ErrPromotionInvalid
	}

	now := time.Now()
	promotions, err := s.promotionRepo.GetAllActiveByProduct(product.ID, now)
	if err != nil {
		return nil, models.Money{}, err
	}
	if len(promotions) == 0 {
		return nil, product.PriceAmount, nil
	}

	subtotal := product.PriceAmount.Decimal.Mul(decimal.NewFromInt(int64(quantity)))

	// ?????? MinAmount,?????? MinAmount <= subtotal ???
	var matched *models.Promotion
	for i := len(promotions) - 1; i >= 0; i-- {
		p := &promotions[i]
		if strings.ToLower(strings.TrimSpace(p.ScopeType)) != constants.ScopeTypeProduct {
			continue
		}
		if p.MinAmount.Decimal.LessThanOrEqual(decimal.Zero) || subtotal.Cmp(p.MinAmount.Decimal) >= 0 {
			matched = p
			break
		}
	}

	if matched == nil {
		return nil, product.PriceAmount, nil
	}

	unitPrice, err := s.calculateUnitPrice(product.PriceAmount, matched)
	if err != nil {
		return nil, models.Money{}, err
	}

	return matched, unitPrice, nil
}

func (s *PromotionService) calculateUnitPrice(base models.Money, promotion *models.Promotion) (models.Money, error) {
	value := promotion.Value.Decimal
	if value.LessThanOrEqual(decimal.Zero) {
		return models.Money{}, ErrPromotionInvalid
	}

	switch strings.ToLower(strings.TrimSpace(promotion.Type)) {
	case constants.PromotionTypeFixed:
		discounted := base.Decimal.Sub(value)
		if discounted.LessThan(decimal.Zero) {
			discounted = decimal.Zero
		}
		return models.NewMoneyFromDecimal(discounted), nil
	case constants.PromotionTypePercent:
		percent := decimal.NewFromInt(100).Sub(value)
		if percent.LessThan(decimal.Zero) {
			percent = decimal.Zero
		}
		discounted := base.Decimal.Mul(percent).Div(decimal.NewFromInt(100))
		return models.NewMoneyFromDecimal(discounted), nil
	case constants.PromotionTypeSpecialPrice:
		return models.NewMoneyFromDecimal(value), nil
	default:
		return models.Money{}, ErrPromotionInvalid
	}
}
