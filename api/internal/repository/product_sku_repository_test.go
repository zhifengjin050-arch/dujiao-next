package repository

import (
	"fmt"
	"testing"
	"time"

	"github.com/dujiao-next/internal/models"
	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func setupProductSKURepositoryTest(t *testing.T) *GormProductSKURepository {
	t.Helper()
	dsn := fmt.Sprintf("file:product_sku_repository_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.ProductSKU{}); err != nil {
		t.Fatalf("migrate product sku failed: %v", err)
	}
	return NewProductSKURepository(db)
}

func TestProductSKURepositoryListByProductSortOrderDescending(t *testing.T) {
	repo := setupProductSKURepositoryTest(t)

	high := &models.ProductSKU{
		ProductID:      1,
		SKUCode:        "HIGH",
		PriceAmount:    models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
		IsActive:       true,
		SortOrder:      100,
		SpecValuesJSON: models.JSON{},
	}
	low := &models.ProductSKU{
		ProductID:      1,
		SKUCode:        "LOW",
		PriceAmount:    models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
		IsActive:       true,
		SortOrder:      1,
		SpecValuesJSON: models.JSON{},
	}
	if err := repo.Create(high); err != nil {
		t.Fatalf("create high sort sku failed: %v", err)
	}
	if err := repo.Create(low); err != nil {
		t.Fatalf("create low sort sku failed: %v", err)
	}

	rows, err := repo.ListByProduct(1, true)
	if err != nil {
		t.Fatalf("list skus failed: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 skus, got %d", len(rows))
	}
	if rows[0].SKUCode != "HIGH" || rows[1].SKUCode != "LOW" {
		t.Fatalf("expected high sort_order first, got %s then %s", rows[0].SKUCode, rows[1].SKUCode)
	}
}
