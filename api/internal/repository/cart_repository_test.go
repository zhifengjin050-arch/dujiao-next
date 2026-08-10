package repository

import (
	"fmt"
	"testing"
	"time"

	"github.com/dujiao-next/internal/models"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupCartRepositoryTest(t *testing.T) (*GormCartRepository, *gorm.DB) {
	t.Helper()
	dsn := fmt.Sprintf("file:cart_repository_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.CartItem{}); err != nil {
		t.Fatalf("migrate cart item failed: %v", err)
	}
	return NewCartRepository(db), db
}

func TestCartRepositoryUpsertUsesProductAndSKUDimension(t *testing.T) {
	repo, db := setupCartRepositoryTest(t)
	now := time.Now()

	first := &models.CartItem{
		UserID:          10001,
		ProductID:       888,
		SKUID:           101,
		Quantity:        1,
		FulfillmentType: "manual",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	second := &models.CartItem{
		UserID:          10001,
		ProductID:       888,
		SKUID:           102,
		Quantity:        2,
		FulfillmentType: "manual",
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := repo.Upsert(first); err != nil {
		t.Fatalf("upsert first sku failed: %v", err)
	}
	if err := repo.Upsert(second); err != nil {
		t.Fatalf("upsert second sku failed: %v", err)
	}

	var count int64
	if err := db.Model(&models.CartItem{}).
		Where("user_id = ? AND product_id = ?", first.UserID, first.ProductID).
		Count(&count).Error; err != nil {
		t.Fatalf("count cart items failed: %v", err)
	}
	if count != 2 {
		t.Fatalf("cart rows want 2 got %d", count)
	}

	first.Quantity = 5
	first.UpdatedAt = now.Add(time.Minute)
	if err := repo.Upsert(first); err != nil {
		t.Fatalf("update first sku failed: %v", err)
	}

	var gotFirst models.CartItem
	if err := db.Where("user_id = ? AND product_id = ? AND sku_id = ?", first.UserID, first.ProductID, first.SKUID).First(&gotFirst).Error; err != nil {
		t.Fatalf("query first sku row failed: %v", err)
	}
	if gotFirst.Quantity != 5 {
		t.Fatalf("first sku quantity want 5 got %d", gotFirst.Quantity)
	}

	var gotSecond models.CartItem
	if err := db.Where("user_id = ? AND product_id = ? AND sku_id = ?", second.UserID, second.ProductID, second.SKUID).First(&gotSecond).Error; err != nil {
		t.Fatalf("query second sku row failed: %v", err)
	}
	if gotSecond.Quantity != 2 {
		t.Fatalf("second sku quantity should keep 2 got %d", gotSecond.Quantity)
	}
}
