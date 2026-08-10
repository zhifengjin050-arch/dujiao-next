package service

import (
	"strconv"
	"strings"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/repository"
)

// CartItemDetail ??????(????)
type CartItemDetail struct {
	ProductID       uint               `json:"product_id"`
	SKUID           uint               `json:"sku_id"`
	Quantity        int                `json:"quantity"`
	FulfillmentType string             `json:"fulfillment_type"`
	UnitPrice       models.Money       `json:"unit_price"`
	OriginalPrice   models.Money       `json:"original_price"`
	Currency        string             `json:"currency"`
	Product         *models.Product    `json:"product"`
	SKU             *models.ProductSKU `json:"sku"`
}

// UpsertCartItemInput ???????
type UpsertCartItemInput struct {
	UserID          uint
	ProductID       uint
	SKUID           uint
	Quantity        int
	FulfillmentType string
}

// CartService ?????
type CartService struct {
	cartRepo       repository.CartRepository
	productRepo    repository.ProductRepository
	productSKURepo repository.ProductSKURepository
	promotionRepo  repository.PromotionRepository
	settingService *SettingService
}

// NewCartService ???????
func NewCartService(cartRepo repository.CartRepository, productRepo repository.ProductRepository, productSKURepo repository.ProductSKURepository, promotionRepo repository.PromotionRepository, settingService *SettingService) *CartService {
	return &CartService{
		cartRepo:       cartRepo,
		productRepo:    productRepo,
		productSKURepo: productSKURepo,
		promotionRepo:  promotionRepo,
		settingService: settingService,
	}
}

// ListByUser ???????
func (s *CartService) ListByUser(userID uint) ([]CartItemDetail, error) {
	if userID == 0 {
		return nil, ErrInvalidOrderItem
	}
	items, err := s.cartRepo.ListByUser(userID)
	if err != nil {
		return nil, err
	}
	currency := resolveServiceSiteCurrency(s.settingService)
	details := make([]CartItemDetail, 0, len(items))
	promotionService := NewPromotionService(s.promotionRepo)
	for _, item := range items {
		product := item.Product
		if product == nil || product.ID == 0 {
			p, err := s.productRepo.GetByID(strconv.FormatUint(uint64(item.ProductID), 10))
			if err != nil {
				return nil, err
			}
			product = p
		}
		if product == nil || !product.IsActive {
			_ = s.cartRepo.DeleteByUserProductSKU(userID, item.ProductID, item.SKUID)
			continue
		}

		sku := item.SKU
		if sku == nil || sku.ID == 0 {
			resolvedSKU, resolveErr := resolveProductOrderSKU(s.productSKURepo, product, item.SKUID)
			if resolveErr != nil {
				_ = s.cartRepo.DeleteByUserProductSKU(userID, item.ProductID, item.SKUID)
				continue
			}
			sku = resolvedSKU
		}

		if sku == nil || !sku.IsActive {
			_ = s.cartRepo.DeleteByUserProductSKU(userID, item.ProductID, item.SKUID)
			continue
		}
		if strings.TrimSpace(product.FulfillmentType) == constants.FulfillmentTypeManual &&
			shouldEnforceManualSKUStock(product, sku) &&
			manualSKUAvailable(sku) <= 0 {
			_ = s.cartRepo.DeleteByUserProductSKU(userID, item.ProductID, item.SKUID)
			continue
		}

		priceCarrier := *product
		priceCarrier.PriceAmount = sku.PriceAmount
		unitPrice := sku.PriceAmount
		if promotionService != nil {
			_, discounted, err := promotionService.ApplyPromotion(&priceCarrier, item.Quantity)
			if err != nil {
				return nil, err
			}
			unitPrice = discounted
		}

		fulfillmentType := strings.TrimSpace(product.FulfillmentType)
		if fulfillmentType == "" {
			fulfillmentType = constants.FulfillmentTypeManual
		}

		details = append(details, CartItemDetail{
			ProductID:       item.ProductID,
			SKUID:           sku.ID,
			Quantity:        item.Quantity,
			FulfillmentType: fulfillmentType,
			UnitPrice:       unitPrice,
			OriginalPrice:   sku.PriceAmount,
			Currency:        currency,
			Product:         product,
			SKU:             sku,
		})
	}
	return details, nil
}

// UpsertItem ?????????
func (s *CartService) UpsertItem(input UpsertCartItemInput) error {
	if input.UserID == 0 || input.ProductID == 0 || input.Quantity <= 0 {
		return ErrInvalidOrderItem
	}
	product, err := s.productRepo.GetByID(strconv.FormatUint(uint64(input.ProductID), 10))
	if err != nil {
		return err
	}
	if product == nil || !product.IsActive {
		return ErrProductNotAvailable
	}
	if err := validateProductPurchaseQuantity(product, input.Quantity); err != nil {
		return err
	}
	sku, err := resolveProductOrderSKU(s.productSKURepo, product, input.SKUID)
	if err != nil {
		return err
	}

	fulfillmentType := strings.TrimSpace(product.FulfillmentType)
	if fulfillmentType == "" {
		fulfillmentType = constants.FulfillmentTypeManual
	}
	if fulfillmentType != constants.FulfillmentTypeManual && fulfillmentType != constants.FulfillmentTypeAuto {
		return ErrFulfillmentInvalid
	}
	if fulfillmentType == constants.FulfillmentTypeManual &&
		shouldEnforceManualSKUStock(product, sku) &&
		manualSKUAvailable(sku) < input.Quantity {
		return ErrManualStockInsufficient
	}

	now := time.Now()
	item := &models.CartItem{
		UserID:          input.UserID,
		ProductID:       input.ProductID,
		SKUID:           sku.ID,
		Quantity:        input.Quantity,
		FulfillmentType: fulfillmentType,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	return s.cartRepo.Upsert(item)
}

// RemoveItem ??????
func (s *CartService) RemoveItem(userID, productID, skuID uint) error {
	if userID == 0 || productID == 0 {
		return ErrInvalidOrderItem
	}
	return s.cartRepo.DeleteByUserProductSKU(userID, productID, skuID)
}
