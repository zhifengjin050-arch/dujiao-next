package service

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/repository"
	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func newSyncSingleSKURepo(t *testing.T) repository.ProductSKURepository {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.ProductSKU{}); err != nil {
		t.Fatalf("auto migrate product sku failed: %v", err)
	}
	return repository.NewProductSKURepository(db)
}

func TestSyncSingleProductSKU_MultipleRowsKeepsSingleActive(t *testing.T) {
	repo := newSyncSingleSKURepo(t)
	productID := uint(2001)

	inactiveDefault := models.ProductSKU{
		ProductID:        productID,
		SKUCode:          models.DefaultSKUCode,
		PriceAmount:      models.NewMoneyFromDecimal(decimal.NewFromInt(10)),
		ManualStockTotal: 9,
		IsActive:         false,
		SortOrder:        0,
	}
	firstActive := models.ProductSKU{
		ProductID:        productID,
		SKUCode:          "A",
		PriceAmount:      models.NewMoneyFromDecimal(decimal.NewFromInt(20)),
		ManualStockTotal: 2,
		IsActive:         true,
		SortOrder:        2,
	}
	secondActive := models.ProductSKU{
		ProductID:        productID,
		SKUCode:          "B",
		PriceAmount:      models.NewMoneyFromDecimal(decimal.NewFromInt(30)),
		ManualStockTotal: 4,
		IsActive:         true,
		SortOrder:        1,
	}
	if err := repo.Create(&inactiveDefault); err != nil {
		t.Fatalf("create inactive default sku failed: %v", err)
	}
	inactiveDefault.IsActive = false
	if err := repo.Update(&inactiveDefault); err != nil {
		t.Fatalf("update inactive default sku failed: %v", err)
	}
	if err := repo.Create(&firstActive); err != nil {
		t.Fatalf("create first active sku failed: %v", err)
	}
	if err := repo.Create(&secondActive); err != nil {
		t.Fatalf("create second active sku failed: %v", err)
	}

	targetPrice := decimal.RequireFromString("88.88")
	if err := syncSingleProductSKU(repo, productID, targetPrice, decimal.Zero, 5, true); err != nil {
		t.Fatalf("sync single sku failed: %v", err)
	}

	skus, err := repo.ListByProduct(productID, false)
	if err != nil {
		t.Fatalf("list sku failed: %v", err)
	}

	activeCount := 0
	for _, sku := range skus {
		if !sku.IsActive {
			continue
		}
		activeCount++
		if sku.ID != firstActive.ID {
			t.Fatalf("expected higher sort_order active sku id=%d, got id=%d", firstActive.ID, sku.ID)
		}
		if !sku.PriceAmount.Equal(targetPrice) {
			t.Fatalf("expected price %s, got %s", targetPrice.StringFixed(2), sku.PriceAmount.String())
		}
		if sku.ManualStockTotal != 5 {
			t.Fatalf("expected manual stock total 5, got %d", sku.ManualStockTotal)
		}
	}
	if activeCount != 1 {
		t.Fatalf("expected exactly one active sku, got %d", activeCount)
	}
}

func TestSyncSingleProductSKU_NoActivePrefersDefaultCode(t *testing.T) {
	repo := newSyncSingleSKURepo(t)
	productID := uint(2002)

	inactiveA := models.ProductSKU{
		ProductID:        productID,
		SKUCode:          "A",
		PriceAmount:      models.NewMoneyFromDecimal(decimal.NewFromInt(10)),
		ManualStockTotal: 3,
		IsActive:         false,
		SortOrder:        1,
	}
	inactiveDefault := models.ProductSKU{
		ProductID:        productID,
		SKUCode:          models.DefaultSKUCode,
		PriceAmount:      models.NewMoneyFromDecimal(decimal.NewFromInt(20)),
		ManualStockTotal: 8,
		IsActive:         false,
		SortOrder:        0,
	}
	if err := repo.Create(&inactiveA); err != nil {
		t.Fatalf("create inactive sku A failed: %v", err)
	}
	inactiveA.IsActive = false
	if err := repo.Update(&inactiveA); err != nil {
		t.Fatalf("update inactive sku A failed: %v", err)
	}
	if err := repo.Create(&inactiveDefault); err != nil {
		t.Fatalf("create inactive default sku failed: %v", err)
	}
	inactiveDefault.IsActive = false
	if err := repo.Update(&inactiveDefault); err != nil {
		t.Fatalf("update inactive default sku failed: %v", err)
	}

	targetPrice := decimal.RequireFromString("19.90")
	if err := syncSingleProductSKU(repo, productID, targetPrice, decimal.Zero, 6, true); err != nil {
		t.Fatalf("sync single sku failed: %v", err)
	}

	skus, err := repo.ListByProduct(productID, false)
	if err != nil {
		t.Fatalf("list sku failed: %v", err)
	}

	activeCount := 0
	for _, sku := range skus {
		if !sku.IsActive {
			continue
		}
		activeCount++
		if sku.ID != inactiveDefault.ID {
			t.Fatalf("expected default sku id=%d to be active, got id=%d", inactiveDefault.ID, sku.ID)
		}
		if !sku.PriceAmount.Equal(targetPrice) {
			t.Fatalf("expected price %s, got %s", targetPrice.StringFixed(2), sku.PriceAmount.String())
		}
		if sku.ManualStockTotal != 6 {
			t.Fatalf("expected manual stock total 6, got %d", sku.ManualStockTotal)
		}
	}
	if activeCount != 1 {
		t.Fatalf("expected exactly one active sku, got %d", activeCount)
	}
}

func TestApplyAutoStockCounts_LegacyStockPrefersDefaultSKU(t *testing.T) {
	svc, db := newAutoStockProductService(t)
	productID := uint(3001)
	defaultSKUID := uint(101)
	otherSKUID := uint(102)

	insertCardSecrets(t, db, productID, 0, models.CardSecretStatusAvailable, 2)
	insertCardSecrets(t, db, productID, 0, models.CardSecretStatusReserved, 1)
	insertCardSecrets(t, db, productID, 0, models.CardSecretStatusUsed, 1)
	insertCardSecrets(t, db, productID, defaultSKUID, models.CardSecretStatusAvailable, 3)
	insertCardSecrets(t, db, productID, otherSKUID, models.CardSecretStatusAvailable, 4)
	counts, err := svc.cardSecretRepo.CountStockByProductIDs([]uint{productID})
	if err != nil {
		t.Fatalf("count stock by product ids failed: %v", err)
	}
	if len(counts) != 5 {
		t.Fatalf("expected 5 grouped stock rows, got %d", len(counts))
	}
	bySKUAndStatus := make(map[uint]map[string]int64)
	for _, row := range counts {
		if bySKUAndStatus[row.SKUID] == nil {
			bySKUAndStatus[row.SKUID] = make(map[string]int64)
		}
		bySKUAndStatus[row.SKUID][row.Status] = row.Total
	}
	if bySKUAndStatus[0][models.CardSecretStatusAvailable] != 2 ||
		bySKUAndStatus[0][models.CardSecretStatusReserved] != 1 ||
		bySKUAndStatus[0][models.CardSecretStatusUsed] != 1 {
		t.Fatalf("unexpected legacy sku(0) rows: %+v", bySKUAndStatus[0])
	}
	if bySKUAndStatus[defaultSKUID][models.CardSecretStatusAvailable] != 3 {
		t.Fatalf("unexpected default sku rows: %+v", bySKUAndStatus[defaultSKUID])
	}
	if bySKUAndStatus[otherSKUID][models.CardSecretStatusAvailable] != 4 {
		t.Fatalf("unexpected other sku rows: %+v", bySKUAndStatus[otherSKUID])
	}

	products := []models.Product{
		{
			ID:              productID,
			FulfillmentType: constants.FulfillmentTypeAuto,
			SKUs: []models.ProductSKU{
				{
					ID:       otherSKUID,
					SKUCode:  "B",
					IsActive: true,
				},
				{
					ID:       defaultSKUID,
					SKUCode:  models.DefaultSKUCode,
					IsActive: true,
				},
			},
		},
	}

	if err := svc.ApplyAutoStockCounts(products); err != nil {
		t.Fatalf("apply auto stock counts failed: %v", err)
	}

	got := products[0]
	if got.AutoStockAvailable != 9 {
		t.Fatalf("expected product auto available=9, got %d", got.AutoStockAvailable)
	}
	if got.AutoStockLocked != 1 {
		t.Fatalf("expected product auto locked=1, got %d", got.AutoStockLocked)
	}
	if got.AutoStockSold != 1 {
		t.Fatalf("expected product auto sold=1, got %d", got.AutoStockSold)
	}
	if got.AutoStockTotal != 10 {
		t.Fatalf("expected product auto total=10, got %d", got.AutoStockTotal)
	}

	if got.SKUs[0].AutoStockAvailable != 4 {
		t.Fatalf("expected other sku auto available=4, got %d", got.SKUs[0].AutoStockAvailable)
	}
	if got.SKUs[0].AutoStockLocked != 0 || got.SKUs[0].AutoStockSold != 0 {
		t.Fatalf("expected other sku locked/sold to remain 0, got locked=%d sold=%d", got.SKUs[0].AutoStockLocked, got.SKUs[0].AutoStockSold)
	}

	if got.SKUs[1].AutoStockAvailable != 5 {
		t.Fatalf("expected default sku auto available=5, got %d", got.SKUs[1].AutoStockAvailable)
	}
	if got.SKUs[1].AutoStockLocked != 1 {
		t.Fatalf("expected default sku auto locked=1, got %d", got.SKUs[1].AutoStockLocked)
	}
	if got.SKUs[1].AutoStockSold != 1 {
		t.Fatalf("expected default sku auto sold=1, got %d", got.SKUs[1].AutoStockSold)
	}
}

func newAutoStockProductService(t *testing.T) (*ProductService, *gorm.DB) {
	t.Helper()

	dsn := fmt.Sprintf("file:product_auto_stock_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.CardSecret{}); err != nil {
		t.Fatalf("auto migrate card secret failed: %v", err)
	}
	secretRepo := repository.NewCardSecretRepository(db)
	return NewProductService(nil, nil, secretRepo, nil, nil, nil, nil, nil, nil), db
}

func insertCardSecrets(t *testing.T, db *gorm.DB, productID, skuID uint, status string, count int) {
	t.Helper()
	for i := 0; i < count; i++ {
		row := models.CardSecret{
			ProductID: productID,
			SKUID:     skuID,
			Secret:    fmt.Sprintf("secret-%d-%d-%s-%d", productID, skuID, status, i),
			Status:    status,
		}
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("create card secret failed: %v", err)
		}
	}
}

func TestProductServiceListPublicIncludesChildProductsForParentCategory(t *testing.T) {
	svc, db := newProductServiceForTest(t)

	parent := models.Category{
		Slug:     "games",
		NameJSON: models.JSON{"zh-CN": "games"},
	}
	child := models.Category{
		ParentID: 1,
		Slug:     "steam",
		NameJSON: models.JSON{"zh-CN": "steam"},
	}
	if err := db.Create(&parent).Error; err != nil {
		t.Fatalf("create parent category failed: %v", err)
	}
	child.ParentID = parent.ID
	if err := db.Create(&child).Error; err != nil {
		t.Fatalf("create child category failed: %v", err)
	}

	parentProduct := models.Product{
		CategoryID:  parent.ID,
		Slug:        "parent-product",
		TitleJSON:   models.JSON{"zh-CN": "parent-product"},
		PriceAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(10)),
		IsActive:    true,
	}
	childProduct := models.Product{
		CategoryID:  child.ID,
		Slug:        "child-product",
		TitleJSON:   models.JSON{"zh-CN": "child-product"},
		PriceAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(10)),
		IsActive:    true,
	}
	if err := db.Create(&parentProduct).Error; err != nil {
		t.Fatalf("create parent product failed: %v", err)
	}
	if err := db.Create(&childProduct).Error; err != nil {
		t.Fatalf("create child product failed: %v", err)
	}

	products, total, err := svc.ListPublic(strconv.FormatUint(uint64(parent.ID), 10), "", 1, 20)
	if err != nil {
		t.Fatalf("list public products failed: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected total=2, got %d", total)
	}
	if len(products) != 2 {
		t.Fatalf("expected 2 products, got %d", len(products))
	}
}

func TestProductServiceCreateRejectsParentCategoryWithChildren(t *testing.T) {
	svc, db := newProductServiceForTest(t)

	parent := models.Category{
		Slug:     "games",
		NameJSON: models.JSON{"zh-CN": "games"},
	}
	if err := db.Create(&parent).Error; err != nil {
		t.Fatalf("create parent category failed: %v", err)
	}
	child := models.Category{
		ParentID: parent.ID,
		Slug:     "steam",
		NameJSON: models.JSON{"zh-CN": "steam"},
	}
	if err := db.Create(&child).Error; err != nil {
		t.Fatalf("create child category failed: %v", err)
	}

	_, err := svc.Create(CreateProductInput{
		CategoryID:      parent.ID,
		Slug:            "invalid-parent-product",
		TitleJSON:       map[string]interface{}{"zh-CN": "invalid-parent-product"},
		PriceAmount:     decimal.NewFromInt(10),
		PurchaseType:    constants.ProductPurchaseMember,
		FulfillmentType: constants.FulfillmentTypeManual,
		ManualStockTotal: func() *int {
			value := 1
			return &value
		}(),
	})
	if err != ErrProductCategoryInvalid {
		t.Fatalf("expected ErrProductCategoryInvalid, got %v", err)
	}
}

func TestProductServiceListPublicSortOrderDescending(t *testing.T) {
	svc, db := newProductServiceForTest(t)

	high := models.Product{
		CategoryID:  1,
		Slug:        "high-sort-product",
		TitleJSON:   models.JSON{"zh-CN": "high"},
		PriceAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(10)),
		IsActive:    true,
		SortOrder:   100,
	}
	low := models.Product{
		CategoryID:  1,
		Slug:        "low-sort-product",
		TitleJSON:   models.JSON{"zh-CN": "low"},
		PriceAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(10)),
		IsActive:    true,
		SortOrder:   1,
	}
	if err := db.Create(&high).Error; err != nil {
		t.Fatalf("create high sort product failed: %v", err)
	}
	if err := db.Create(&low).Error; err != nil {
		t.Fatalf("create low sort product failed: %v", err)
	}

	rows, total, err := svc.ListPublic("", "", 1, 20)
	if err != nil {
		t.Fatalf("list public products failed: %v", err)
	}
	if total != 2 {
		t.Fatalf("expected total=2, got %d", total)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 products, got %d", len(rows))
	}
	if rows[0].Slug != "high-sort-product" || rows[1].Slug != "low-sort-product" {
		t.Fatalf("expected high sort_order first, got %s then %s", rows[0].Slug, rows[1].Slug)
	}
}

func TestProductServiceListPublicSortsSKUsDescending(t *testing.T) {
	svc, db := newProductServiceForTest(t)

	product := models.Product{
		CategoryID:  1,
		Slug:        "sku-order-product",
		TitleJSON:   models.JSON{"zh-CN": "sku-order-product"},
		PriceAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(10)),
		IsActive:    true,
		SortOrder:   0,
	}
	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("create product failed: %v", err)
	}

	high := models.ProductSKU{
		ProductID:      product.ID,
		SKUCode:        "HIGH",
		SpecValuesJSON: models.JSON{},
		PriceAmount:    models.NewMoneyFromDecimal(decimal.NewFromInt(10)),
		IsActive:       true,
		SortOrder:      100,
	}
	low := models.ProductSKU{
		ProductID:      product.ID,
		SKUCode:        "LOW",
		SpecValuesJSON: models.JSON{},
		PriceAmount:    models.NewMoneyFromDecimal(decimal.NewFromInt(10)),
		IsActive:       true,
		SortOrder:      1,
	}
	if err := db.Create(&high).Error; err != nil {
		t.Fatalf("create high sort sku failed: %v", err)
	}
	if err := db.Create(&low).Error; err != nil {
		t.Fatalf("create low sort sku failed: %v", err)
	}

	rows, total, err := svc.ListPublic("", "", 1, 20)
	if err != nil {
		t.Fatalf("list public products failed: %v", err)
	}
	if total != 1 || len(rows) != 1 {
		t.Fatalf("expected exactly 1 product, total=%d len=%d", total, len(rows))
	}
	if len(rows[0].SKUs) != 2 {
		t.Fatalf("expected 2 skus, got %d", len(rows[0].SKUs))
	}
	if rows[0].SKUs[0].SKUCode != "HIGH" || rows[0].SKUs[1].SKUCode != "LOW" {
		t.Fatalf("expected high sort_order sku first, got %s then %s", rows[0].SKUs[0].SKUCode, rows[0].SKUs[1].SKUCode)
	}
}

func TestProductServiceGetAdminByIDIncludesInactiveSKUs(t *testing.T) {
	svc, db := newProductServiceForTest(t)

	product := models.Product{
		CategoryID:  1,
		Slug:        "admin-all-skus-product",
		TitleJSON:   models.JSON{"zh-CN": "admin-all-skus-product"},
		PriceAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(10)),
		IsActive:    true,
	}
	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("create product failed: %v", err)
	}

	activeSKU := models.ProductSKU{
		ProductID:      product.ID,
		SKUCode:        "ACTIVE",
		SpecValuesJSON: models.JSON{},
		PriceAmount:    models.NewMoneyFromDecimal(decimal.NewFromInt(10)),
		IsActive:       true,
		SortOrder:      10,
	}
	inactiveSKU := models.ProductSKU{
		ProductID:      product.ID,
		SKUCode:        "INACTIVE",
		SpecValuesJSON: models.JSON{},
		PriceAmount:    models.NewMoneyFromDecimal(decimal.NewFromInt(20)),
		IsActive:       false,
		SortOrder:      1,
	}
	if err := db.Create(&activeSKU).Error; err != nil {
		t.Fatalf("create active sku failed: %v", err)
	}
	if err := db.Create(&inactiveSKU).Error; err != nil {
		t.Fatalf("create inactive sku failed: %v", err)
	}
	inactiveSKU.IsActive = false
	if err := db.Save(&inactiveSKU).Error; err != nil {
		t.Fatalf("persist inactive sku failed: %v", err)
	}

	got, err := svc.GetAdminByID(strconv.FormatUint(uint64(product.ID), 10))
	if err != nil {
		t.Fatalf("get admin product failed: %v", err)
	}
	if got == nil {
		t.Fatal("expected product, got nil")
	}
	if len(got.SKUs) != 2 {
		t.Fatalf("expected 2 skus for admin detail, got %d", len(got.SKUs))
	}
	if got.SKUs[0].SKUCode != "ACTIVE" || !got.SKUs[0].IsActive {
		t.Fatalf("expected first sku to be active ACTIVE, got %+v", got.SKUs[0])
	}
	if got.SKUs[1].SKUCode != "INACTIVE" || got.SKUs[1].IsActive {
		t.Fatalf("expected second sku to be inactive INACTIVE, got %+v", got.SKUs[1])
	}
}

func TestProductServiceUpdateKeepsMappedProductFulfillmentUpstream(t *testing.T) {
	svc, db := newProductServiceForTest(t)

	category := models.Category{
		Slug:     "mapped-category",
		NameJSON: models.JSON{"zh-CN": "mapped-category"},
	}
	if err := db.Create(&category).Error; err != nil {
		t.Fatalf("create category failed: %v", err)
	}

	product := models.Product{
		CategoryID:       category.ID,
		Slug:             "mapped-product",
		TitleJSON:        models.JSON{"zh-CN": "mapped-product"},
		PriceAmount:      models.NewMoneyFromDecimal(decimal.NewFromInt(10)),
		PurchaseType:     constants.ProductPurchaseMember,
		FulfillmentType:  constants.FulfillmentTypeUpstream,
		ManualStockTotal: 0,
		IsMapped:         true,
		IsActive:         true,
	}
	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("create mapped product failed: %v", err)
	}

	sku := models.ProductSKU{
		ProductID:      product.ID,
		SKUCode:        models.DefaultSKUCode,
		SpecValuesJSON: models.JSON{},
		PriceAmount:    models.NewMoneyFromDecimal(decimal.NewFromInt(10)),
		IsActive:       true,
	}
	if err := db.Create(&sku).Error; err != nil {
		t.Fatalf("create mapped product sku failed: %v", err)
	}

	updated, err := svc.Update(strconv.FormatUint(uint64(product.ID), 10), CreateProductInput{
		CategoryID:      category.ID,
		Slug:            "mapped-product-updated",
		TitleJSON:       map[string]interface{}{"zh-CN": "mapped-product-updated"},
		PriceAmount:     decimal.NewFromInt(20),
		PurchaseType:    constants.ProductPurchaseMember,
		FulfillmentType: constants.FulfillmentTypeAuto,
		ManualStockTotal: func() *int {
			value := 0
			return &value
		}(),
		IsActive: func() *bool {
			value := true
			return &value
		}(),
	})
	if err != nil {
		t.Fatalf("update mapped product failed: %v", err)
	}
	if updated.FulfillmentType != constants.FulfillmentTypeUpstream {
		t.Fatalf("expected mapped product fulfillment type to remain upstream, got %s", updated.FulfillmentType)
	}

	reloaded, err := svc.GetAdminByID(strconv.FormatUint(uint64(product.ID), 10))
	if err != nil {
		t.Fatalf("reload mapped product failed: %v", err)
	}
	if reloaded.FulfillmentType != constants.FulfillmentTypeUpstream {
		t.Fatalf("expected persisted fulfillment type upstream, got %s", reloaded.FulfillmentType)
	}
}

func TestProductServiceUpdateRejectsDisablingAutoSKUWithCardSecretStock(t *testing.T) {
	svc, db := newProductServiceForTest(t)

	category := models.Category{
		Slug:     "auto-card-secret-category",
		NameJSON: models.JSON{"zh-CN": "auto-card-secret-category"},
	}
	if err := db.Create(&category).Error; err != nil {
		t.Fatalf("create category failed: %v", err)
	}

	product := models.Product{
		CategoryID:      category.ID,
		Slug:            "auto-card-secret-product",
		TitleJSON:       models.JSON{"zh-CN": "auto-card-secret-product"},
		PriceAmount:     models.NewMoneyFromDecimal(decimal.NewFromInt(10)),
		PurchaseType:    constants.ProductPurchaseMember,
		FulfillmentType: constants.FulfillmentTypeAuto,
		IsActive:        true,
	}
	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("create product failed: %v", err)
	}

	stockSKU := models.ProductSKU{
		ProductID:      product.ID,
		SKUCode:        "SKU-STOCK",
		SpecValuesJSON: models.JSON{"zh-CN": "???"},
		PriceAmount:    models.NewMoneyFromDecimal(decimal.NewFromInt(10)),
		IsActive:       true,
		SortOrder:      2,
	}
	spareSKU := models.ProductSKU{
		ProductID:      product.ID,
		SKUCode:        "SKU-SPARE",
		SpecValuesJSON: models.JSON{"zh-CN": "???"},
		PriceAmount:    models.NewMoneyFromDecimal(decimal.NewFromInt(10)),
		IsActive:       true,
		SortOrder:      1,
	}
	if err := db.Create(&stockSKU).Error; err != nil {
		t.Fatalf("create stock sku failed: %v", err)
	}
	if err := db.Create(&spareSKU).Error; err != nil {
		t.Fatalf("create spare sku failed: %v", err)
	}

	insertCardSecrets(t, db, product.ID, stockSKU.ID, models.CardSecretStatusAvailable, 1)

	_, err := svc.Update(strconv.FormatUint(uint64(product.ID), 10), CreateProductInput{
		CategoryID:      category.ID,
		Slug:            product.Slug,
		TitleJSON:       map[string]interface{}{"zh-CN": "auto-card-secret-product"},
		PriceAmount:     decimal.NewFromInt(10),
		PurchaseType:    constants.ProductPurchaseMember,
		FulfillmentType: constants.FulfillmentTypeAuto,
		SKUs: []ProductSKUInput{
			{
				ID:             stockSKU.ID,
				SKUCode:        stockSKU.SKUCode,
				SpecValuesJSON: map[string]interface{}{"zh-CN": "???"},
				PriceAmount:    decimal.NewFromInt(10),
				IsActive: func() *bool {
					value := false
					return &value
				}(),
				SortOrder: 2,
			},
			{
				ID:             spareSKU.ID,
				SKUCode:        spareSKU.SKUCode,
				SpecValuesJSON: map[string]interface{}{"zh-CN": "???"},
				PriceAmount:    decimal.NewFromInt(10),
				IsActive: func() *bool {
					value := true
					return &value
				}(),
				SortOrder: 1,
			},
		},
		IsActive: func() *bool {
			value := true
			return &value
		}(),
	})
	if err != ErrProductSKUHasCardSecretStock {
		t.Fatalf("update product error want %v got %v", ErrProductSKUHasCardSecretStock, err)
	}
}

func newProductServiceForTest(t *testing.T) (*ProductService, *gorm.DB) {
	t.Helper()

	dsn := fmt.Sprintf("file:product_service_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.Category{}, &models.Product{}, &models.ProductSKU{}, &models.CardSecret{}, &models.CardSecretBatch{}, &models.MemberLevelPrice{}, &models.CartItem{}, &models.ProductMapping{}, &models.SKUMapping{}, &models.Order{}, &models.OrderItem{}); err != nil {
		t.Fatalf("auto migrate product service tables failed: %v", err)
	}

	return NewProductService(
		repository.NewProductRepository(db),
		repository.NewProductSKURepository(db),
		repository.NewCardSecretRepository(db),
		repository.NewCardSecretBatchRepository(db),
		repository.NewCategoryRepository(db),
		repository.NewMemberLevelPriceRepository(db),
		repository.NewCartRepository(db),
		repository.NewProductMappingRepository(db),
		repository.NewOrderRepository(db),
	), db
}

func TestProductServiceDeleteCascade(t *testing.T) {
	svc, db := newProductServiceForTest(t)

	// ????
	cat := models.Category{Slug: "test-cat", NameJSON: models.JSON{"zh-CN": "test"}}
	if err := db.Create(&cat).Error; err != nil {
		t.Fatalf("create category: %v", err)
	}

	// ????
	product := models.Product{
		CategoryID:      cat.ID,
		Slug:            "test-product",
		TitleJSON:       models.JSON{"zh-CN": "test-product"},
		PriceAmount:     models.NewMoneyFromDecimal(decimal.NewFromInt(10)),
		FulfillmentType: constants.FulfillmentTypeManual,
		IsActive:        true,
	}
	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("create product: %v", err)
	}
	productID := strconv.FormatUint(uint64(product.ID), 10)

	// ???? SKU
	sku := models.ProductSKU{
		ProductID:   product.ID,
		SKUCode:     "DEFAULT",
		PriceAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(10)),
		IsActive:    true,
	}
	if err := db.Create(&sku).Error; err != nil {
		t.Fatalf("create sku: %v", err)
	}

	// ????????
	mlp := models.MemberLevelPrice{
		ProductID:     product.ID,
		SKUID:         sku.ID,
		MemberLevelID: 1,
		PriceAmount:   models.NewMoneyFromDecimal(decimal.NewFromInt(8)),
	}
	if err := db.Create(&mlp).Error; err != nil {
		t.Fatalf("create member level price: %v", err)
	}

	// ??????
	cart := models.CartItem{
		UserID:          1,
		ProductID:       product.ID,
		SKUID:           sku.ID,
		Quantity:        1,
		FulfillmentType: constants.FulfillmentTypeManual,
	}
	if err := db.Create(&cart).Error; err != nil {
		t.Fatalf("create cart item: %v", err)
	}

	// ??????
	pm := models.ProductMapping{
		ConnectionID:      1,
		LocalProductID:    product.ID,
		UpstreamProductID: 100,
	}
	if err := db.Create(&pm).Error; err != nil {
		t.Fatalf("create product mapping: %v", err)
	}

	// ?? SKU ??
	sm := models.SKUMapping{
		ProductMappingID: pm.ID,
		LocalSKUID:       sku.ID,
		UpstreamSKUID:    200,
	}
	if err := db.Create(&sm).Error; err != nil {
		t.Fatalf("create sku mapping: %v", err)
	}

	// ????
	if err := svc.Delete(productID); err != nil {
		t.Fatalf("delete product: %v", err)
	}

	// ?????????????
	var skuCount int64
	db.Model(&models.ProductSKU{}).Where("product_id = ?", product.ID).Count(&skuCount)
	if skuCount != 0 {
		t.Errorf("expected 0 SKUs after delete, got %d", skuCount)
	}

	var mlpCount int64
	db.Model(&models.MemberLevelPrice{}).Where("product_id = ?", product.ID).Count(&mlpCount)
	if mlpCount != 0 {
		t.Errorf("expected 0 member level prices after delete, got %d", mlpCount)
	}

	var cartCount int64
	db.Model(&models.CartItem{}).Where("product_id = ?", product.ID).Count(&cartCount)
	if cartCount != 0 {
		t.Errorf("expected 0 cart items after delete, got %d", cartCount)
	}

	var pmCount int64
	db.Model(&models.ProductMapping{}).Where("local_product_id = ?", product.ID).Count(&pmCount)
	if pmCount != 0 {
		t.Errorf("expected 0 product mappings after delete, got %d", pmCount)
	}

	var smCount int64
	db.Model(&models.SKUMapping{}).Where("product_mapping_id = ?", pm.ID).Count(&smCount)
	if smCount != 0 {
		t.Errorf("expected 0 SKU mappings after delete, got %d", smCount)
	}

	// ???????????
	var productCount int64
	db.Model(&models.Product{}).Where("id = ?", product.ID).Count(&productCount)
	if productCount != 0 {
		t.Errorf("expected product to be soft-deleted, but still found %d", productCount)
	}
}
