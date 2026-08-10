package public

import (
	"fmt"
	"testing"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/repository"
	"github.com/dujiao-next/internal/service"

	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func TestDecoratePublicProductDisplayPricePrefersFirstActiveSKU(t *testing.T) {
	h := &Handler{}
	product := &models.Product{
		ID:          1,
		PriceAmount: models.NewMoneyFromDecimal(decimal.RequireFromString("59.90")),
		SKUs: []models.ProductSKU{
			{
				ID:          11,
				IsActive:    true,
				SortOrder:   100,
				PriceAmount: models.NewMoneyFromDecimal(decimal.RequireFromString("89.90")),
			},
			{
				ID:          12,
				IsActive:    true,
				SortOrder:   10,
				PriceAmount: models.NewMoneyFromDecimal(decimal.RequireFromString("49.90")),
			},
		},
	}

	item, err := h.decoratePublicProduct(product, nil)
	if err != nil {
		t.Fatalf("decoratePublicProduct failed: %v", err)
	}
	expected := decimal.RequireFromString("89.90")
	if !item.PriceAmount.Decimal.Equal(expected) {
		t.Fatalf("expected display price %s, got: %s", expected.String(), item.PriceAmount.String())
	}
}

func TestDecoratePublicProductPromotionUsesDisplayPrice(t *testing.T) {
	dsn := fmt.Sprintf("file:public_product_display_price_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.Promotion{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	promotion := models.Promotion{
		Name:       "fixed-10",
		ScopeType:  constants.ScopeTypeProduct,
		ScopeRefID: 1,
		Type:       constants.PromotionTypeFixed,
		Value:      models.NewMoneyFromDecimal(decimal.NewFromInt(10)),
		MinAmount:  models.NewMoneyFromDecimal(decimal.Zero),
		IsActive:   true,
	}
	if err := db.Create(&promotion).Error; err != nil {
		t.Fatalf("create promotion failed: %v", err)
	}

	h := &Handler{}
	product := &models.Product{
		ID:          1,
		PriceAmount: models.NewMoneyFromDecimal(decimal.RequireFromString("59.90")),
		SKUs: []models.ProductSKU{
			{
				ID:          21,
				IsActive:    true,
				SortOrder:   100,
				PriceAmount: models.NewMoneyFromDecimal(decimal.RequireFromString("89.90")),
			},
			{
				ID:          22,
				IsActive:    true,
				SortOrder:   10,
				PriceAmount: models.NewMoneyFromDecimal(decimal.RequireFromString("49.90")),
			},
		},
	}

	promoService := service.NewPromotionService(repository.NewPromotionRepository(db))
	item, err := h.decoratePublicProduct(product, promoService)
	if err != nil {
		t.Fatalf("decoratePublicProduct failed: %v", err)
	}
	if item.PromotionPriceAmount == nil {
		t.Fatalf("expected promotion price amount")
	}

	expectedDisplay := decimal.RequireFromString("89.90")
	expectedPromotion := decimal.RequireFromString("79.90")
	if !item.PriceAmount.Decimal.Equal(expectedDisplay) {
		t.Fatalf("expected display price %s, got: %s", expectedDisplay.String(), item.PriceAmount.String())
	}
	if !item.PromotionPriceAmount.Decimal.Equal(expectedPromotion) {
		t.Fatalf("expected promotion display price %s, got: %s", expectedPromotion.String(), item.PromotionPriceAmount.String())
	}
}
