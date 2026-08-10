package models

import (
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func setupSKUMigrationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:sku_migration_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	DB = db
	return db
}

func TestEnsureProductSKUMigrationBackfillLegacyData(t *testing.T) {
	db := setupSKUMigrationTestDB(t)

	if err := db.AutoMigrate(
		&Product{},
		&ProductSKU{},
		&OrderItem{},
		&CartItem{},
		&CardSecret{},
		&CardSecretBatch{},
	); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	product := &Product{
		CategoryID:        1,
		Slug:              "sku-migration-legacy",
		TitleJSON:         JSON{"zh-CN": "????"},
		PriceAmount:       NewMoneyFromDecimal(decimal.NewFromInt(128)),
		PurchaseType:      "member",
		FulfillmentType:   "manual",
		ManualStockTotal:  20,
		ManualStockLocked: 3,
		ManualStockSold:   5,
		IsActive:          true,
	}
	if err := db.Create(product).Error; err != nil {
		t.Fatalf("create product failed: %v", err)
	}

	now := time.Now()
	orderItem := &OrderItem{
		OrderID:         1,
		ProductID:       product.ID,
		SKUID:           0,
		TitleJSON:       JSON{"zh-CN": "????"},
		UnitPrice:       product.PriceAmount,
		Quantity:        1,
		TotalPrice:      product.PriceAmount,
		FulfillmentType: "manual",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := db.Create(orderItem).Error; err != nil {
		t.Fatalf("create order item failed: %v", err)
	}

	cartItem := &CartItem{
		UserID:          1001,
		ProductID:       product.ID,
		SKUID:           0,
		Quantity:        2,
		FulfillmentType: "manual",
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := db.Create(cartItem).Error; err != nil {
		t.Fatalf("create cart item failed: %v", err)
	}

	batch := &CardSecretBatch{
		ProductID:  product.ID,
		SKUID:      0,
		BatchNo:    "SKU-MIGRATION-BATCH-001",
		Source:     "manual",
		TotalCount: 1,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if err := db.Create(batch).Error; err != nil {
		t.Fatalf("create card secret batch failed: %v", err)
	}

	secret := &CardSecret{
		ProductID: product.ID,
		SKUID:     0,
		BatchID:   &batch.ID,
		Secret:    "CARD-001",
		Status:    CardSecretStatusAvailable,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := db.Create(secret).Error; err != nil {
		t.Fatalf("create card secret failed: %v", err)
	}

	if err := ensureProductSKUMigration(); err != nil {
		t.Fatalf("ensure sku migration failed: %v", err)
	}

	var sku ProductSKU
	if err := db.Where("product_id = ? AND sku_code = ?", product.ID, DefaultSKUCode).First(&sku).Error; err != nil {
		t.Fatalf("query default sku failed: %v", err)
	}
	if !sku.PriceAmount.Decimal.Equal(product.PriceAmount.Decimal) {
		t.Fatalf("default sku price mismatch want %s got %s", product.PriceAmount.String(), sku.PriceAmount.String())
	}
	if sku.ManualStockTotal != product.ManualStockTotal || sku.ManualStockLocked != product.ManualStockLocked || sku.ManualStockSold != product.ManualStockSold {
		t.Fatalf("default sku stock snapshot mismatch")
	}

	var gotOrderItem OrderItem
	if err := db.First(&gotOrderItem, orderItem.ID).Error; err != nil {
		t.Fatalf("reload order item failed: %v", err)
	}
	if gotOrderItem.SKUID != sku.ID {
		t.Fatalf("order item sku_id want %d got %d", sku.ID, gotOrderItem.SKUID)
	}

	var gotCartItem CartItem
	if err := db.First(&gotCartItem, cartItem.ID).Error; err != nil {
		t.Fatalf("reload cart item failed: %v", err)
	}
	if gotCartItem.SKUID != sku.ID {
		t.Fatalf("cart item sku_id want %d got %d", sku.ID, gotCartItem.SKUID)
	}

	var gotBatch CardSecretBatch
	if err := db.First(&gotBatch, batch.ID).Error; err != nil {
		t.Fatalf("reload card secret batch failed: %v", err)
	}
	if gotBatch.SKUID != sku.ID {
		t.Fatalf("card secret batch sku_id want %d got %d", sku.ID, gotBatch.SKUID)
	}

	var gotSecret CardSecret
	if err := db.First(&gotSecret, secret.ID).Error; err != nil {
		t.Fatalf("reload card secret failed: %v", err)
	}
	if gotSecret.SKUID != sku.ID {
		t.Fatalf("card secret sku_id want %d got %d", sku.ID, gotSecret.SKUID)
	}

	if err := ensureProductSKUMigration(); err != nil {
		t.Fatalf("ensure sku migration second run failed: %v", err)
	}

	var skuCount int64
	if err := db.Model(&ProductSKU{}).Where("product_id = ?", product.ID).Count(&skuCount).Error; err != nil {
		t.Fatalf("count product sku failed: %v", err)
	}
	if skuCount != 1 {
		t.Fatalf("idempotent check failed: sku count want 1 got %d", skuCount)
	}
}

func TestMigrateCartSKUUniqueIndex(t *testing.T) {
	db := setupSKUMigrationTestDB(t)
	if err := db.AutoMigrate(&CartItem{}); err != nil {
		t.Fatalf("auto migrate cart item failed: %v", err)
	}

	// ????????,???????????????????
	if err := db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_cart_user_product ON cart_items(user_id, product_id)").Error; err != nil {
		t.Fatalf("create legacy index failed: %v", err)
	}

	if err := migrateCartSKUUniqueIndex(); err != nil {
		t.Fatalf("migrate cart unique index failed: %v", err)
	}

	if db.Migrator().HasIndex(&CartItem{}, "idx_cart_user_product") {
		t.Fatalf("legacy unique index idx_cart_user_product should be dropped")
	}
	if !db.Migrator().HasIndex(&CartItem{}, "idx_cart_user_product_sku") {
		t.Fatalf("new unique index idx_cart_user_product_sku should exist")
	}
}
