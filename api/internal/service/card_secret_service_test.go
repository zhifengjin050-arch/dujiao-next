package service

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/repository"

	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func setupCardSecretServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:card_secret_service_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(
		&models.Product{},
		&models.ProductSKU{},
		&models.CardSecretBatch{},
		&models.CardSecret{},
	); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}
	models.DB = db
	return db
}

func TestCreateCardSecretBatchAutoMultiSKURequiresExplicitSKU(t *testing.T) {
	db := setupCardSecretServiceTestDB(t)

	product := &models.Product{
		CategoryID:      1,
		Slug:            "card-secret-product-default",
		TitleJSON:       models.JSON{"zh-CN": "????"},
		PriceAmount:     models.NewMoneyFromDecimal(decimal.NewFromInt(20)),
		PurchaseType:    constants.ProductPurchaseMember,
		FulfillmentType: constants.FulfillmentTypeAuto,
		IsActive:        true,
	}
	if err := db.Create(product).Error; err != nil {
		t.Fatalf("create product failed: %v", err)
	}

	defaultSKU := &models.ProductSKU{
		ProductID:   product.ID,
		SKUCode:     models.DefaultSKUCode,
		PriceAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(20)),
		IsActive:    true,
	}
	if err := db.Create(defaultSKU).Error; err != nil {
		t.Fatalf("create default sku failed: %v", err)
	}
	otherSKU := &models.ProductSKU{
		ProductID:   product.ID,
		SKUCode:     "PRO",
		PriceAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(30)),
		IsActive:    true,
	}
	if err := db.Create(otherSKU).Error; err != nil {
		t.Fatalf("create other sku failed: %v", err)
	}

	svc := NewCardSecretService(
		repository.NewCardSecretRepository(db),
		repository.NewCardSecretBatchRepository(db),
		repository.NewProductRepository(db),
		repository.NewProductSKURepository(db),
	)

	batch, created, err := svc.CreateCardSecretBatch(CreateCardSecretBatchInput{
		ProductID: product.ID,
		Secrets:   []string{"AAA-001", "AAA-002"},
		Source:    constants.CardSecretSourceManual,
		AdminID:   1,
	})
	if err != ErrProductSKURequired {
		t.Fatalf("create card secret batch error want %v got %v", ErrProductSKURequired, err)
	}
	if batch != nil || created != 0 {
		t.Fatalf("batch should not be created when sku is omitted for auto multi-sku product")
	}
}

func TestCreateCardSecretBatchAutoSingleActiveFallsBackToOnlyActiveSKU(t *testing.T) {
	db := setupCardSecretServiceTestDB(t)

	product := &models.Product{
		CategoryID:      1,
		Slug:            "card-secret-product-single-active",
		TitleJSON:       models.JSON{"zh-CN": "????"},
		PriceAmount:     models.NewMoneyFromDecimal(decimal.NewFromInt(20)),
		PurchaseType:    constants.ProductPurchaseMember,
		FulfillmentType: constants.FulfillmentTypeAuto,
		IsActive:        true,
	}
	if err := db.Create(product).Error; err != nil {
		t.Fatalf("create product failed: %v", err)
	}

	defaultSKU := &models.ProductSKU{
		ProductID:   product.ID,
		SKUCode:     models.DefaultSKUCode,
		PriceAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(20)),
		IsActive:    false,
	}
	if err := db.Create(defaultSKU).Error; err != nil {
		t.Fatalf("create default sku failed: %v", err)
	}
	if err := db.Model(defaultSKU).Update("is_active", false).Error; err != nil {
		t.Fatalf("disable default sku failed: %v", err)
	}
	onlyActiveSKU := &models.ProductSKU{
		ProductID:   product.ID,
		SKUCode:     "PRO",
		PriceAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(30)),
		IsActive:    true,
	}
	if err := db.Create(onlyActiveSKU).Error; err != nil {
		t.Fatalf("create active sku failed: %v", err)
	}

	svc := NewCardSecretService(
		repository.NewCardSecretRepository(db),
		repository.NewCardSecretBatchRepository(db),
		repository.NewProductRepository(db),
		repository.NewProductSKURepository(db),
	)

	batch, created, err := svc.CreateCardSecretBatch(CreateCardSecretBatchInput{
		ProductID: product.ID,
		Secrets:   []string{"AAA-101", "AAA-102"},
		Source:    constants.CardSecretSourceManual,
		AdminID:   1,
	})
	if err != nil {
		t.Fatalf("create card secret batch failed: %v", err)
	}
	if created != 2 {
		t.Fatalf("created count want 2 got %d", created)
	}
	if batch.SKUID != onlyActiveSKU.ID {
		t.Fatalf("batch sku_id want active %d got %d", onlyActiveSKU.ID, batch.SKUID)
	}
}

func TestCardSecretServiceSupportsBatchTargetOperations(t *testing.T) {
	db := setupCardSecretServiceTestDB(t)

	product := &models.Product{
		CategoryID:      1,
		Slug:            "card-secret-batch-ops",
		TitleJSON:       models.JSON{"zh-CN": "??????"},
		PriceAmount:     models.NewMoneyFromDecimal(decimal.NewFromInt(50)),
		PurchaseType:    constants.ProductPurchaseMember,
		FulfillmentType: constants.FulfillmentTypeAuto,
		IsActive:        true,
	}
	if err := db.Create(product).Error; err != nil {
		t.Fatalf("create product failed: %v", err)
	}

	defaultSKU := &models.ProductSKU{
		ProductID:   product.ID,
		SKUCode:     models.DefaultSKUCode,
		PriceAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(50)),
		IsActive:    true,
	}
	if err := db.Create(defaultSKU).Error; err != nil {
		t.Fatalf("create default sku failed: %v", err)
	}

	secretRepo := repository.NewCardSecretRepository(db)
	svc := NewCardSecretService(
		secretRepo,
		repository.NewCardSecretBatchRepository(db),
		repository.NewProductRepository(db),
		repository.NewProductSKURepository(db),
	)

	batchA, created, err := svc.CreateCardSecretBatch(CreateCardSecretBatchInput{
		ProductID: product.ID,
		Secrets:   []string{"BATCH-A-001", "BATCH-A-002"},
		Source:    constants.CardSecretSourceManual,
	})
	if err != nil {
		t.Fatalf("create batch A failed: %v", err)
	}
	if created != 2 {
		t.Fatalf("batch A created want 2 got %d", created)
	}

	batchB, created, err := svc.CreateCardSecretBatch(CreateCardSecretBatchInput{
		ProductID: product.ID,
		Secrets:   []string{"BATCH-B-001"},
		Source:    constants.CardSecretSourceManual,
	})
	if err != nil {
		t.Fatalf("create batch B failed: %v", err)
	}
	if created != 1 {
		t.Fatalf("batch B created want 1 got %d", created)
	}

	rows, total, err := svc.ListCardSecrets(ListCardSecretInput{
		ProductID: product.ID,
		BatchID:   batchA.ID,
		Page:      1,
		PageSize:  20,
	})
	if err != nil {
		t.Fatalf("list card secrets by batch failed: %v", err)
	}
	if total != 2 || len(rows) != 2 {
		t.Fatalf("list by batch A want total=2 len=2 got total=%d len=%d", total, len(rows))
	}
	for _, row := range rows {
		if row.BatchID == nil || *row.BatchID != batchA.ID {
			t.Fatalf("expected batch A id %d got %+v", batchA.ID, row.BatchID)
		}
	}

	affected, err := svc.BatchUpdateCardSecretStatus(nil, batchA.ID, ListCardSecretInput{}, models.CardSecretStatusUsed)
	if err != nil {
		t.Fatalf("batch update status by batch id failed: %v", err)
	}
	if affected != 2 {
		t.Fatalf("batch update affected want 2 got %d", affected)
	}

	batchAIDs, err := secretRepo.ListIDsByBatchID(batchA.ID)
	if err != nil {
		t.Fatalf("list batch A ids failed: %v", err)
	}
	batchASecrets, err := secretRepo.ListByIDs(batchAIDs)
	if err != nil {
		t.Fatalf("list batch A secrets failed: %v", err)
	}
	for _, row := range batchASecrets {
		if row.Status != models.CardSecretStatusUsed {
			t.Fatalf("batch A status want used got %s", row.Status)
		}
	}

	batchBIDs, err := secretRepo.ListIDsByBatchID(batchB.ID)
	if err != nil {
		t.Fatalf("list batch B ids failed: %v", err)
	}
	batchBSecrets, err := secretRepo.ListByIDs(batchBIDs)
	if err != nil {
		t.Fatalf("list batch B secrets failed: %v", err)
	}
	if len(batchBSecrets) != 1 || batchBSecrets[0].Status != models.CardSecretStatusAvailable {
		t.Fatalf("batch B status should remain available, got %+v", batchBSecrets)
	}

	content, contentType, err := svc.ExportCardSecrets(nil, batchA.ID, ListCardSecretInput{}, constants.ExportFormatTXT)
	if err != nil {
		t.Fatalf("export batch A secrets failed: %v", err)
	}
	if contentType != "text/plain; charset=utf-8" {
		t.Fatalf("export content type mismatch: %s", contentType)
	}
	exported := string(content)
	if !strings.Contains(exported, "BATCH-A-001") || !strings.Contains(exported, "BATCH-A-002") {
		t.Fatalf("exported content missing batch A secrets: %s", exported)
	}
	if strings.Contains(exported, "BATCH-B-001") {
		t.Fatalf("exported content should not contain batch B secret: %s", exported)
	}

	deleted, err := svc.BatchDeleteCardSecrets(nil, batchB.ID, ListCardSecretInput{})
	if err != nil {
		t.Fatalf("delete batch B secrets failed: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("delete batch B affected want 1 got %d", deleted)
	}

	batchBIDs, err = secretRepo.ListIDsByBatchID(batchB.ID)
	if err != nil {
		t.Fatalf("reload batch B ids failed: %v", err)
	}
	if len(batchBIDs) != 0 {
		t.Fatalf("batch B ids want empty got %v", batchBIDs)
	}
}

func TestCardSecretServiceSupportsKeywordAndBatchNoFilters(t *testing.T) {
	db := setupCardSecretServiceTestDB(t)

	product := &models.Product{
		CategoryID:      1,
		Slug:            "card-secret-search",
		TitleJSON:       models.JSON{"zh-CN": "??????"},
		PriceAmount:     models.NewMoneyFromDecimal(decimal.NewFromInt(30)),
		PurchaseType:    constants.ProductPurchaseMember,
		FulfillmentType: constants.FulfillmentTypeAuto,
		IsActive:        true,
	}
	if err := db.Create(product).Error; err != nil {
		t.Fatalf("create product failed: %v", err)
	}

	defaultSKU := &models.ProductSKU{
		ProductID:   product.ID,
		SKUCode:     models.DefaultSKUCode,
		PriceAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(30)),
		IsActive:    true,
	}
	if err := db.Create(defaultSKU).Error; err != nil {
		t.Fatalf("create default sku failed: %v", err)
	}

	svc := NewCardSecretService(
		repository.NewCardSecretRepository(db),
		repository.NewCardSecretBatchRepository(db),
		repository.NewProductRepository(db),
		repository.NewProductSKURepository(db),
	)

	if _, _, err := svc.CreateCardSecretBatch(CreateCardSecretBatchInput{
		ProductID: product.ID,
		Secrets:   []string{"AAA-SEARCH-001", "AAA-SEARCH-002"},
		BatchNo:   "BATCH-SEARCH-A",
		Source:    constants.CardSecretSourceManual,
	}); err != nil {
		t.Fatalf("create batch A failed: %v", err)
	}
	if _, _, err := svc.CreateCardSecretBatch(CreateCardSecretBatchInput{
		ProductID: product.ID,
		Secrets:   []string{"BBB-KEEP-001"},
		BatchNo:   "BATCH-SEARCH-B",
		Source:    constants.CardSecretSourceManual,
	}); err != nil {
		t.Fatalf("create batch B failed: %v", err)
	}

	items, total, err := svc.ListCardSecrets(ListCardSecretInput{
		ProductID: product.ID,
		Secret:    "SEARCH-002",
		Page:      1,
		PageSize:  20,
	})
	if err != nil {
		t.Fatalf("filter by secret failed: %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].Secret != "AAA-SEARCH-002" {
		t.Fatalf("filter by secret mismatch, total=%d items=%+v", total, items)
	}

	items, total, err = svc.ListCardSecrets(ListCardSecretInput{
		ProductID: product.ID,
		BatchNo:   "SEARCH-A",
		Page:      1,
		PageSize:  20,
	})
	if err != nil {
		t.Fatalf("filter by batch no failed: %v", err)
	}
	if total != 2 || len(items) != 2 {
		t.Fatalf("filter by batch no want total=2 len=2 got total=%d len=%d", total, len(items))
	}
}

func TestCardSecretServiceListBatchesReturnsRealtimeCounts(t *testing.T) {
	db := setupCardSecretServiceTestDB(t)

	product := &models.Product{
		CategoryID:      1,
		Slug:            "card-secret-batch-summary",
		TitleJSON:       models.JSON{"zh-CN": "????????"},
		PriceAmount:     models.NewMoneyFromDecimal(decimal.NewFromInt(88)),
		PurchaseType:    constants.ProductPurchaseMember,
		FulfillmentType: constants.FulfillmentTypeAuto,
		IsActive:        true,
	}
	if err := db.Create(product).Error; err != nil {
		t.Fatalf("create product failed: %v", err)
	}

	defaultSKU := &models.ProductSKU{
		ProductID:   product.ID,
		SKUCode:     models.DefaultSKUCode,
		PriceAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(88)),
		IsActive:    true,
	}
	if err := db.Create(defaultSKU).Error; err != nil {
		t.Fatalf("create default sku failed: %v", err)
	}

	svc := NewCardSecretService(
		repository.NewCardSecretRepository(db),
		repository.NewCardSecretBatchRepository(db),
		repository.NewProductRepository(db),
		repository.NewProductSKURepository(db),
	)

	batchA, _, err := svc.CreateCardSecretBatch(CreateCardSecretBatchInput{
		ProductID: product.ID,
		Secrets:   []string{"SUMMARY-A-001", "SUMMARY-A-002"},
		BatchNo:   "SUMMARY-A",
		Source:    constants.CardSecretSourceManual,
	})
	if err != nil {
		t.Fatalf("create batch A failed: %v", err)
	}
	batchB, _, err := svc.CreateCardSecretBatch(CreateCardSecretBatchInput{
		ProductID: product.ID,
		Secrets:   []string{"SUMMARY-B-001"},
		BatchNo:   "SUMMARY-B",
		Source:    constants.CardSecretSourceManual,
	})
	if err != nil {
		t.Fatalf("create batch B failed: %v", err)
	}

	rows, err := repository.NewCardSecretRepository(db).ListIDs(repository.CardSecretListFilter{
		ProductID: product.ID,
		BatchID:   batchA.ID,
	})
	if err != nil {
		t.Fatalf("list batch A ids failed: %v", err)
	}
	if _, err := svc.BatchUpdateCardSecretStatus(rows[:1], 0, ListCardSecretInput{}, models.CardSecretStatusReserved); err != nil {
		t.Fatalf("mark batch A reserved failed: %v", err)
	}
	if _, err := svc.BatchUpdateCardSecretStatus(rows[1:], 0, ListCardSecretInput{}, models.CardSecretStatusUsed); err != nil {
		t.Fatalf("mark batch A used failed: %v", err)
	}
	if _, err := svc.BatchDeleteCardSecrets(nil, batchB.ID, ListCardSecretInput{}); err != nil {
		t.Fatalf("delete batch B failed: %v", err)
	}

	summaries, total, err := svc.ListBatches(product.ID, defaultSKU.ID, 1, 20)
	if err != nil {
		t.Fatalf("list batches failed: %v", err)
	}
	if total != 2 || len(summaries) != 2 {
		t.Fatalf("list batches want total=2 len=2 got total=%d len=%d", total, len(summaries))
	}

	for _, summary := range summaries {
		switch summary.BatchNo {
		case "SUMMARY-A":
			if summary.TotalCount != 2 || summary.AvailableCount != 0 || summary.ReservedCount != 1 || summary.UsedCount != 1 {
				t.Fatalf("summary A mismatch: %+v", summary)
			}
		case "SUMMARY-B":
			if summary.TotalCount != 0 || summary.AvailableCount != 0 || summary.ReservedCount != 0 || summary.UsedCount != 0 {
				t.Fatalf("summary B mismatch: %+v", summary)
			}
		default:
			t.Fatalf("unexpected batch summary: %+v", summary)
		}
	}
}
