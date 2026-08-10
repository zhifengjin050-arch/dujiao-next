package repository

import (
	"fmt"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"

	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func setupDashboardRepositoryTest(t *testing.T) (*GormDashboardRepository, *gorm.DB) {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Category{}, &models.Product{}, &models.Order{}, &models.OrderItem{}); err != nil {
		t.Fatalf("migrate dashboard models failed: %v", err)
	}
	if err := db.AutoMigrate(&models.ProductSKU{}); err != nil {
		t.Fatalf("migrate dashboard sku models failed: %v", err)
	}
	if err := db.AutoMigrate(&models.PaymentChannel{}, &models.Payment{}, &models.OrderRefundRecord{}); err != nil {
		t.Fatalf("migrate dashboard models failed: %v", err)
	}
	return NewDashboardRepository(db), db
}

func createDashboardCategory(t *testing.T, db *gorm.DB, slug string) *models.Category {
	t.Helper()
	category := &models.Category{
		Slug:     slug,
		NameJSON: models.JSON{"zh-CN": "????"},
	}
	if err := db.Create(category).Error; err != nil {
		t.Fatalf("create category failed: %v", err)
	}
	return category
}

func TestGetTopProductsIncludesChildOrderItems(t *testing.T) {
	repo, db := setupDashboardRepositoryTest(t)
	now := time.Now()

	category := createDashboardCategory(t, db, "test-category")

	product := &models.Product{
		CategoryID:      category.ID,
		Slug:            "test-dashboard-product",
		TitleJSON:       models.JSON{"zh-CN": "????"},
		PriceAmount:     models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
		PurchaseType:    constants.ProductPurchaseMember,
		FulfillmentType: constants.FulfillmentTypeManual,
		IsActive:        true,
	}
	if err := db.Create(product).Error; err != nil {
		t.Fatalf("create product failed: %v", err)
	}

	parentOrder := &models.Order{
		OrderNo:        "DJ-TEST-PARENT",
		UserID:         1,
		Status:         constants.OrderStatusPaid,
		Currency:       "CNY",
		OriginalAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
		DiscountAmount: models.NewMoneyFromDecimal(decimal.Zero),
		TotalAmount:    models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
		CreatedAt:      now,
	}
	if err := db.Create(parentOrder).Error; err != nil {
		t.Fatalf("create parent order failed: %v", err)
	}

	childOrder := &models.Order{
		OrderNo:        "DJ-TEST-PARENT-01",
		ParentID:       &parentOrder.ID,
		UserID:         1,
		Status:         constants.OrderStatusPaid,
		Currency:       "CNY",
		OriginalAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
		DiscountAmount: models.NewMoneyFromDecimal(decimal.Zero),
		TotalAmount:    models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
		CreatedAt:      now,
	}
	if err := db.Create(childOrder).Error; err != nil {
		t.Fatalf("create child order failed: %v", err)
	}

	orderItem := &models.OrderItem{
		OrderID:           childOrder.ID,
		ProductID:         product.ID,
		TitleJSON:         models.JSON{"zh-CN": "????"},
		UnitPrice:         models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
		Quantity:          2,
		TotalPrice:        models.NewMoneyFromDecimal(decimal.NewFromInt(200)),
		CouponDiscount:    models.NewMoneyFromDecimal(decimal.NewFromInt(10)),
		PromotionDiscount: models.NewMoneyFromDecimal(decimal.NewFromInt(20)),
		FulfillmentType:   constants.FulfillmentTypeManual,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := db.Create(orderItem).Error; err != nil {
		t.Fatalf("create order item failed: %v", err)
	}

	rows, err := repo.GetTopProducts(now.Add(-time.Hour), now.Add(time.Hour), 5)
	if err != nil {
		t.Fatalf("get top products failed: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows len want 1 got %d", len(rows))
	}
	if rows[0].ProductID != product.ID {
		t.Fatalf("product id want %d got %d", product.ID, rows[0].ProductID)
	}
	if rows[0].PaidOrders != 1 {
		t.Fatalf("paid orders want 1 got %d", rows[0].PaidOrders)
	}
	if rows[0].Quantity != 2 {
		t.Fatalf("quantity want 2 got %d", rows[0].Quantity)
	}
	if rows[0].PaidAmount != 190 {
		t.Fatalf("paid amount want 190 got %.2f", rows[0].PaidAmount)
	}
}

func TestPaymentStatsExcludeWalletProvider(t *testing.T) {
	repo, db := setupDashboardRepositoryTest(t)
	now := time.Now().UTC().Truncate(time.Second)

	channel := &models.PaymentChannel{
		Name:            "???",
		ProviderType:    constants.PaymentProviderOfficial,
		ChannelType:     constants.PaymentChannelTypeAlipay,
		InteractionMode: constants.PaymentInteractionRedirect,
		FeeRate:         models.NewMoneyFromDecimal(decimal.Zero),
		IsActive:        true,
	}
	if err := db.Create(channel).Error; err != nil {
		t.Fatalf("create channel failed: %v", err)
	}

	onlineSuccess := &models.Payment{
		OrderID:         1,
		ChannelID:       channel.ID,
		ProviderType:    constants.PaymentProviderOfficial,
		ChannelType:     constants.PaymentChannelTypeAlipay,
		InteractionMode: constants.PaymentInteractionRedirect,
		Amount:          models.NewMoneyFromDecimal(decimal.NewFromInt(120)),
		FeeRate:         models.NewMoneyFromDecimal(decimal.Zero),
		FeeAmount:       models.NewMoneyFromDecimal(decimal.Zero),
		Currency:        "CNY",
		Status:          constants.PaymentStatusSuccess,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := db.Create(onlineSuccess).Error; err != nil {
		t.Fatalf("create online success payment failed: %v", err)
	}

	onlineFailed := &models.Payment{
		OrderID:         2,
		ChannelID:       channel.ID,
		ProviderType:    constants.PaymentProviderOfficial,
		ChannelType:     constants.PaymentChannelTypeAlipay,
		InteractionMode: constants.PaymentInteractionRedirect,
		Amount:          models.NewMoneyFromDecimal(decimal.NewFromInt(88)),
		FeeRate:         models.NewMoneyFromDecimal(decimal.Zero),
		FeeAmount:       models.NewMoneyFromDecimal(decimal.Zero),
		Currency:        "CNY",
		Status:          constants.PaymentStatusFailed,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := db.Create(onlineFailed).Error; err != nil {
		t.Fatalf("create online failed payment failed: %v", err)
	}

	walletSuccess := &models.Payment{
		OrderID:         3,
		ChannelID:       0,
		ProviderType:    constants.PaymentProviderWallet,
		ChannelType:     constants.PaymentChannelTypeBalance,
		InteractionMode: constants.PaymentInteractionBalance,
		Amount:          models.NewMoneyFromDecimal(decimal.NewFromInt(59)),
		FeeRate:         models.NewMoneyFromDecimal(decimal.Zero),
		FeeAmount:       models.NewMoneyFromDecimal(decimal.Zero),
		Currency:        "CNY",
		Status:          constants.PaymentStatusSuccess,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := db.Create(walletSuccess).Error; err != nil {
		t.Fatalf("create wallet payment failed: %v", err)
	}

	startAt := now.Add(-time.Hour)
	endAt := now.Add(time.Hour)

	overview, err := repo.GetOverview(startAt, endAt)
	if err != nil {
		t.Fatalf("get overview failed: %v", err)
	}
	if overview.PaymentsTotal != 2 {
		t.Fatalf("payments total want 2 got %d", overview.PaymentsTotal)
	}
	if overview.PaymentsSuccess != 1 {
		t.Fatalf("payments success want 1 got %d", overview.PaymentsSuccess)
	}
	if overview.PaymentsFailed != 1 {
		t.Fatalf("payments failed want 1 got %d", overview.PaymentsFailed)
	}

	trends, err := repo.GetPaymentTrends(startAt, endAt)
	if err != nil {
		t.Fatalf("get payment trends failed: %v", err)
	}
	if len(trends) == 0 {
		t.Fatalf("payment trends should not be empty")
	}
	point := trends[0]
	if point.PaymentsSuccess != 1 {
		t.Fatalf("trend payments success want 1 got %d", point.PaymentsSuccess)
	}
	if point.PaymentsFailed != 1 {
		t.Fatalf("trend payments failed want 1 got %d", point.PaymentsFailed)
	}
	if point.GMVPaid != 120 {
		t.Fatalf("trend paid amount want 120 got %.2f", point.GMVPaid)
	}

	topChannels, err := repo.GetTopChannels(startAt, endAt, 5)
	if err != nil {
		t.Fatalf("get top channels failed: %v", err)
	}
	if len(topChannels) != 1 {
		t.Fatalf("top channels len want 1 got %d", len(topChannels))
	}
	if topChannels[0].ProviderType != constants.PaymentProviderOfficial {
		t.Fatalf("top channel provider want %s got %s", constants.PaymentProviderOfficial, topChannels[0].ProviderType)
	}
}

func TestGetStockStatsUsesActiveManualSKUs(t *testing.T) {
	repo, db := setupDashboardRepositoryTest(t)
	category := createDashboardCategory(t, db, "dashboard-manual-stock")

	lowStockProduct := &models.Product{
		CategoryID:       category.ID,
		Slug:             "manual-low-stock",
		TitleJSON:        models.JSON{"zh-CN": "? SKU ????"},
		PriceAmount:      models.NewMoneyFromDecimal(decimal.NewFromInt(99)),
		PurchaseType:     constants.ProductPurchaseMember,
		FulfillmentType:  constants.FulfillmentTypeManual,
		ManualStockTotal: 999,
		IsActive:         true,
	}
	if err := db.Create(lowStockProduct).Error; err != nil {
		t.Fatalf("create low stock product failed: %v", err)
	}
	for idx, sku := range []models.ProductSKU{
		{ProductID: lowStockProduct.ID, SKUCode: "A", PriceAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(99)), ManualStockTotal: 2, IsActive: true},
		{ProductID: lowStockProduct.ID, SKUCode: "B", PriceAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(99)), ManualStockTotal: 3, IsActive: true},
		{ProductID: lowStockProduct.ID, SKUCode: "DISABLED", PriceAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(99)), ManualStockTotal: 100, IsActive: false},
	} {
		row := sku
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("create manual sku failed: %v", err)
		}
		if idx == 2 {
			if err := db.Model(&row).Update("is_active", false).Error; err != nil {
				t.Fatalf("disable manual sku failed: %v", err)
			}
		}
	}

	unlimitedProduct := &models.Product{
		CategoryID:       category.ID,
		Slug:             "manual-unlimited-sku",
		TitleJSON:        models.JSON{"zh-CN": "??????"},
		PriceAmount:      models.NewMoneyFromDecimal(decimal.NewFromInt(88)),
		PurchaseType:     constants.ProductPurchaseMember,
		FulfillmentType:  constants.FulfillmentTypeManual,
		ManualStockTotal: 0,
		IsActive:         true,
	}
	if err := db.Create(unlimitedProduct).Error; err != nil {
		t.Fatalf("create unlimited product failed: %v", err)
	}
	unlimitedSKU := &models.ProductSKU{
		ProductID:        unlimitedProduct.ID,
		SKUCode:          "UNLIMITED",
		PriceAmount:      models.NewMoneyFromDecimal(decimal.NewFromInt(88)),
		ManualStockTotal: constants.ManualStockUnlimited,
		IsActive:         true,
	}
	if err := db.Create(unlimitedSKU).Error; err != nil {
		t.Fatalf("create unlimited sku failed: %v", err)
	}

	outOfStockProduct := &models.Product{
		CategoryID:       category.ID,
		Slug:             "manual-fallback-zero",
		TitleJSON:        models.JSON{"zh-CN": "???????"},
		PriceAmount:      models.NewMoneyFromDecimal(decimal.NewFromInt(77)),
		PurchaseType:     constants.ProductPurchaseMember,
		FulfillmentType:  constants.FulfillmentTypeManual,
		ManualStockTotal: 0,
		IsActive:         true,
	}
	if err := db.Create(outOfStockProduct).Error; err != nil {
		t.Fatalf("create fallback product failed: %v", err)
	}

	stats, err := repo.GetStockStats(5)
	if err != nil {
		t.Fatalf("get stock stats failed: %v", err)
	}
	if stats.ManualAvailableUnits != 5 {
		t.Fatalf("manual available units want 5 got %d", stats.ManualAvailableUnits)
	}
	if stats.LowStockProducts != 1 {
		t.Fatalf("low stock products want 1 got %d", stats.LowStockProducts)
	}
	if stats.OutOfStockProducts != 1 {
		t.Fatalf("out of stock products want 1 got %d", stats.OutOfStockProducts)
	}
}

func TestGetInventoryAlertItemsFallsBackToProductLevelWhenOnlyInactiveAutoSKUHasStock(t *testing.T) {
	repo, db := setupDashboardRepositoryTest(t)
	if err := db.AutoMigrate(&models.CardSecret{}); err != nil {
		t.Fatalf("migrate card secret failed: %v", err)
	}

	category := createDashboardCategory(t, db, "dashboard-auto-legacy-stock")
	product := &models.Product{
		CategoryID:      category.ID,
		Slug:            "dashboard-auto-legacy-stock",
		TitleJSON:       models.JSON{"zh-CN": "??????"},
		PriceAmount:     models.NewMoneyFromDecimal(decimal.NewFromInt(99)),
		PurchaseType:    constants.ProductPurchaseMember,
		FulfillmentType: constants.FulfillmentTypeAuto,
		IsActive:        true,
	}
	if err := db.Create(product).Error; err != nil {
		t.Fatalf("create product failed: %v", err)
	}

	legacySKU := &models.ProductSKU{
		ProductID:   product.ID,
		SKUCode:     models.DefaultSKUCode,
		PriceAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(99)),
		IsActive:    false,
		SortOrder:   0,
	}
	activeSKU := &models.ProductSKU{
		ProductID:   product.ID,
		SKUCode:     "SKU-2",
		PriceAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(99)),
		IsActive:    true,
		SortOrder:   1,
	}
	if err := db.Create(legacySKU).Error; err != nil {
		t.Fatalf("create legacy sku failed: %v", err)
	}
	if err := db.Model(legacySKU).Update("is_active", false).Error; err != nil {
		t.Fatalf("disable legacy sku failed: %v", err)
	}
	if err := db.Create(activeSKU).Error; err != nil {
		t.Fatalf("create active sku failed: %v", err)
	}

	for idx := 0; idx < 2; idx++ {
		row := &models.CardSecret{
			ProductID: product.ID,
			SKUID:     legacySKU.ID,
			Secret:    fmt.Sprintf("LEGACY-%d", idx),
			Status:    models.CardSecretStatusAvailable,
		}
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("create legacy card secret failed: %v", err)
		}
	}

	rows, err := repo.GetInventoryAlertItems(5)
	if err != nil {
		t.Fatalf("get inventory alert items failed: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("inventory alert rows want 1 got %d: %+v", len(rows), rows)
	}
	if rows[0].SKUID != activeSKU.ID {
		t.Fatalf("fallback row should reuse the only active sku %d, got skuid=%d", activeSKU.ID, rows[0].SKUID)
	}
	if rows[0].AvailableStock != 2 {
		t.Fatalf("fallback row stock want 2 got %d", rows[0].AvailableStock)
	}
	if rows[0].AlertType != constants.NotificationAlertTypeLowStockProducts {
		t.Fatalf("fallback row alert type want low_stock_products got %s", rows[0].AlertType)
	}
}

func TestGetOverviewUsesOrderCreationWindowForPaidGMV(t *testing.T) {
	repo, db := setupDashboardRepositoryTest(t)
	now := time.Now().UTC().Truncate(time.Second)

	paidOutsideWindow := now.Add(24 * time.Hour)
	inWindowOrder := &models.Order{
		OrderNo:        "DJ-GMV-IN-WINDOW",
		UserID:         1,
		Status:         constants.OrderStatusPaid,
		Currency:       "CNY",
		OriginalAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
		DiscountAmount: models.NewMoneyFromDecimal(decimal.Zero),
		TotalAmount:    models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
		CreatedAt:      now,
		PaidAt:         &paidOutsideWindow,
	}
	if err := db.Create(inWindowOrder).Error; err != nil {
		t.Fatalf("create in-window order failed: %v", err)
	}

	paidInsideWindow := now
	outOfWindowOrder := &models.Order{
		OrderNo:        "DJ-GMV-OUT-WINDOW",
		UserID:         1,
		Status:         constants.OrderStatusPaid,
		Currency:       "CNY",
		OriginalAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(60)),
		DiscountAmount: models.NewMoneyFromDecimal(decimal.Zero),
		TotalAmount:    models.NewMoneyFromDecimal(decimal.NewFromInt(60)),
		CreatedAt:      now.Add(-48 * time.Hour),
		PaidAt:         &paidInsideWindow,
	}
	if err := db.Create(outOfWindowOrder).Error; err != nil {
		t.Fatalf("create out-of-window order failed: %v", err)
	}

	overview, err := repo.GetOverview(now.Add(-time.Hour), now.Add(time.Hour))
	if err != nil {
		t.Fatalf("get overview failed: %v", err)
	}
	if overview.GMVPaid != 100 {
		t.Fatalf("gmv paid want 100 got %.2f", overview.GMVPaid)
	}
}

func TestTrendQueriesBucketByRequestedTimezone(t *testing.T) {
	repo, db := setupDashboardRepositoryTest(t)
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("load location failed: %v", err)
	}
	baseUTC := time.Date(2026, 3, 1, 15, 30, 0, 0, time.UTC)
	nextUTC := time.Date(2026, 3, 1, 16, 30, 0, 0, time.UTC)

	for idx, createdAt := range []time.Time{baseUTC, nextUTC} {
		order := &models.Order{
			OrderNo:        fmt.Sprintf("DJ-TZ-%d", idx),
			UserID:         1,
			Status:         constants.OrderStatusPaid,
			Currency:       "CNY",
			OriginalAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(50)),
			DiscountAmount: models.NewMoneyFromDecimal(decimal.Zero),
			TotalAmount:    models.NewMoneyFromDecimal(decimal.NewFromInt(50)),
			CreatedAt:      createdAt,
		}
		if err := db.Create(order).Error; err != nil {
			t.Fatalf("create order failed: %v", err)
		}
	}

	channel := &models.PaymentChannel{
		Name:            "???",
		ProviderType:    constants.PaymentProviderOfficial,
		ChannelType:     constants.PaymentChannelTypeAlipay,
		InteractionMode: constants.PaymentInteractionRedirect,
		FeeRate:         models.NewMoneyFromDecimal(decimal.Zero),
		IsActive:        true,
	}
	if err := db.Create(channel).Error; err != nil {
		t.Fatalf("create channel failed: %v", err)
	}
	for idx, item := range []struct {
		createdAt time.Time
		status    string
		amount    int64
	}{
		{createdAt: baseUTC, status: constants.PaymentStatusSuccess, amount: 30},
		{createdAt: nextUTC, status: constants.PaymentStatusFailed, amount: 40},
	} {
		payment := &models.Payment{
			OrderID:         uint(idx + 1),
			ChannelID:       channel.ID,
			ProviderType:    constants.PaymentProviderOfficial,
			ChannelType:     constants.PaymentChannelTypeAlipay,
			InteractionMode: constants.PaymentInteractionRedirect,
			Amount:          models.NewMoneyFromDecimal(decimal.NewFromInt(item.amount)),
			FeeRate:         models.NewMoneyFromDecimal(decimal.Zero),
			FeeAmount:       models.NewMoneyFromDecimal(decimal.Zero),
			Currency:        "CNY",
			Status:          item.status,
			CreatedAt:       item.createdAt,
			UpdatedAt:       item.createdAt,
		}
		if err := db.Create(payment).Error; err != nil {
			t.Fatalf("create payment failed: %v", err)
		}
	}

	startAt := time.Date(2026, 3, 1, 0, 0, 0, 0, location)
	endAt := time.Date(2026, 3, 3, 0, 0, 0, 0, location)

	orderRows, err := repo.GetOrderTrends(startAt, endAt)
	if err != nil {
		t.Fatalf("get order trends failed: %v", err)
	}
	if len(orderRows) != 2 {
		t.Fatalf("order trend rows want 2 got %d", len(orderRows))
	}
	if orderRows[0].Day != "2026-03-01" || orderRows[0].OrdersTotal != 1 || orderRows[0].OrdersPaid != 1 {
		t.Fatalf("unexpected first order trend row: %+v", orderRows[0])
	}
	if orderRows[1].Day != "2026-03-02" || orderRows[1].OrdersTotal != 1 || orderRows[1].OrdersPaid != 1 {
		t.Fatalf("unexpected second order trend row: %+v", orderRows[1])
	}

	paymentRows, err := repo.GetPaymentTrends(startAt, endAt)
	if err != nil {
		t.Fatalf("get payment trends failed: %v", err)
	}
	if len(paymentRows) != 2 {
		t.Fatalf("payment trend rows want 2 got %d", len(paymentRows))
	}
	if paymentRows[0].Day != "2026-03-01" || paymentRows[0].PaymentsSuccess != 1 || paymentRows[0].PaymentsFailed != 0 || paymentRows[0].GMVPaid != 30 {
		t.Fatalf("unexpected first payment trend row: %+v", paymentRows[0])
	}
	if paymentRows[1].Day != "2026-03-02" || paymentRows[1].PaymentsSuccess != 0 || paymentRows[1].PaymentsFailed != 1 || paymentRows[1].GMVPaid != 0 {
		t.Fatalf("unexpected second payment trend row: %+v", paymentRows[1])
	}
}

func TestGetProfitOverviewDeductsRefundRecords(t *testing.T) {
	repo, db := setupDashboardRepositoryTest(t)
	now := time.Now().UTC().Truncate(time.Second)

	category := createDashboardCategory(t, db, "dashboard-profit-refund-category")
	product := &models.Product{
		CategoryID:      category.ID,
		Slug:            "dashboard-profit-refund-product",
		TitleJSON:       models.JSON{"zh-CN": "??????"},
		PriceAmount:     models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
		PurchaseType:    constants.ProductPurchaseMember,
		FulfillmentType: constants.FulfillmentTypeManual,
		IsActive:        true,
	}
	if err := db.Create(product).Error; err != nil {
		t.Fatalf("create product failed: %v", err)
	}

	createOrderWithItem := func(orderNo, status string, amount, cost int64, createdAt time.Time) *models.Order {
		t.Helper()
		order := &models.Order{
			OrderNo:        orderNo,
			UserID:         1,
			Status:         status,
			Currency:       "CNY",
			OriginalAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(amount)),
			DiscountAmount: models.NewMoneyFromDecimal(decimal.Zero),
			TotalAmount:    models.NewMoneyFromDecimal(decimal.NewFromInt(amount)),
			CreatedAt:      createdAt,
			UpdatedAt:      createdAt,
		}
		if err := db.Create(order).Error; err != nil {
			t.Fatalf("create order failed: %v", err)
		}
		item := &models.OrderItem{
			OrderID:         order.ID,
			ProductID:       product.ID,
			TitleJSON:       models.JSON{"zh-CN": "??????"},
			UnitPrice:       models.NewMoneyFromDecimal(decimal.NewFromInt(amount)),
			CostPrice:       models.NewMoneyFromDecimal(decimal.NewFromInt(cost)),
			Quantity:        1,
			TotalPrice:      models.NewMoneyFromDecimal(decimal.NewFromInt(amount)),
			CouponDiscount:  models.NewMoneyFromDecimal(decimal.Zero),
			FulfillmentType: constants.FulfillmentTypeManual,
			CreatedAt:       createdAt,
			UpdatedAt:       createdAt,
		}
		if err := db.Create(item).Error; err != nil {
			t.Fatalf("create order item failed: %v", err)
		}
		return order
	}

	manualRefundedOrder := createOrderWithItem("DJ-PROFIT-MANUAL", constants.OrderStatusRefunded, 100, 40, now)
	walletRefundedOrder := createOrderWithItem("DJ-PROFIT-WALLET", constants.OrderStatusPartiallyRefunded, 120, 50, now)

	records := []models.OrderRefundRecord{
		{
			UserID:    1,
			OrderID:   manualRefundedOrder.ID,
			Type:      constants.OrderRefundTypeManual,
			Amount:    models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
			Currency:  "CNY",
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			UserID:    1,
			OrderID:   walletRefundedOrder.ID,
			Type:      constants.OrderRefundTypeWallet,
			Amount:    models.NewMoneyFromDecimal(decimal.NewFromInt(20)),
			Currency:  "CNY",
			CreatedAt: now,
			UpdatedAt: now,
		},
		{
			UserID:    1,
			OrderID:   walletRefundedOrder.ID,
			Type:      constants.OrderRefundTypeManual,
			Amount:    models.NewMoneyFromDecimal(decimal.NewFromInt(10)),
			Currency:  "CNY",
			CreatedAt: now,
			UpdatedAt: now,
		},
	}
	for idx := range records {
		if err := db.Create(&records[idx]).Error; err != nil {
			t.Fatalf("create refund record failed: %v", err)
		}
	}

	result, err := repo.GetProfitOverview(now.Add(-time.Hour), now.Add(time.Hour))
	if err != nil {
		t.Fatalf("get profit overview failed: %v", err)
	}
	if math.Abs(result.TotalRevenue-90) > 0.000001 {
		t.Fatalf("total revenue want 90 got %.2f", result.TotalRevenue)
	}
	if math.Abs(result.TotalCost-90) > 0.000001 {
		t.Fatalf("total cost want 90 got %.2f", result.TotalCost)
	}
}

func TestGetProfitTrendsDeductsRefundRecords(t *testing.T) {
	repo, db := setupDashboardRepositoryTest(t)
	base := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)

	category := createDashboardCategory(t, db, "dashboard-profit-trend-refund-category")
	product := &models.Product{
		CategoryID:      category.ID,
		Slug:            "dashboard-profit-trend-refund-product",
		TitleJSON:       models.JSON{"zh-CN": "????????"},
		PriceAmount:     models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
		PurchaseType:    constants.ProductPurchaseMember,
		FulfillmentType: constants.FulfillmentTypeManual,
		IsActive:        true,
	}
	if err := db.Create(product).Error; err != nil {
		t.Fatalf("create product failed: %v", err)
	}

	createOrderWithItem := func(orderNo, status string, amount, cost int64, createdAt time.Time) *models.Order {
		t.Helper()
		order := &models.Order{
			OrderNo:        orderNo,
			UserID:         1,
			Status:         status,
			Currency:       "CNY",
			OriginalAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(amount)),
			DiscountAmount: models.NewMoneyFromDecimal(decimal.Zero),
			TotalAmount:    models.NewMoneyFromDecimal(decimal.NewFromInt(amount)),
			CreatedAt:      createdAt,
			UpdatedAt:      createdAt,
		}
		if err := db.Create(order).Error; err != nil {
			t.Fatalf("create order failed: %v", err)
		}
		item := &models.OrderItem{
			OrderID:         order.ID,
			ProductID:       product.ID,
			TitleJSON:       models.JSON{"zh-CN": "????????"},
			UnitPrice:       models.NewMoneyFromDecimal(decimal.NewFromInt(amount)),
			CostPrice:       models.NewMoneyFromDecimal(decimal.NewFromInt(cost)),
			Quantity:        1,
			TotalPrice:      models.NewMoneyFromDecimal(decimal.NewFromInt(amount)),
			CouponDiscount:  models.NewMoneyFromDecimal(decimal.Zero),
			FulfillmentType: constants.FulfillmentTypeManual,
			CreatedAt:       createdAt,
			UpdatedAt:       createdAt,
		}
		if err := db.Create(item).Error; err != nil {
			t.Fatalf("create order item failed: %v", err)
		}
		return order
	}

	day1Order := createOrderWithItem("DJ-PROFIT-TREND-DAY1", constants.OrderStatusRefunded, 80, 30, base)
	day2Order := createOrderWithItem("DJ-PROFIT-TREND-DAY2", constants.OrderStatusRefunded, 100, 40, base.Add(24*time.Hour))

	records := []models.OrderRefundRecord{
		{
			UserID:    1,
			OrderID:   day1Order.ID,
			Type:      constants.OrderRefundTypeManual,
			Amount:    models.NewMoneyFromDecimal(decimal.NewFromInt(80)),
			Currency:  "CNY",
			CreatedAt: base,
			UpdatedAt: base,
		},
		{
			UserID:    1,
			OrderID:   day2Order.ID,
			Type:      constants.OrderRefundTypeWallet,
			Amount:    models.NewMoneyFromDecimal(decimal.NewFromInt(30)),
			Currency:  "CNY",
			CreatedAt: base.Add(24 * time.Hour),
			UpdatedAt: base.Add(24 * time.Hour),
		},
		{
			UserID:    1,
			OrderID:   day2Order.ID,
			Type:      constants.OrderRefundTypeManual,
			Amount:    models.NewMoneyFromDecimal(decimal.NewFromInt(10)),
			Currency:  "CNY",
			CreatedAt: base.Add(24 * time.Hour),
			UpdatedAt: base.Add(24 * time.Hour),
		},
	}
	for idx := range records {
		if err := db.Create(&records[idx]).Error; err != nil {
			t.Fatalf("create refund record failed: %v", err)
		}
	}

	rows, err := repo.GetProfitTrends(base.Add(-time.Hour), base.Add(48*time.Hour))
	if err != nil {
		t.Fatalf("get profit trends failed: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("profit trend rows want 2 got %d", len(rows))
	}
	rowMap := make(map[string]DashboardProfitTrendRow, len(rows))
	for _, row := range rows {
		rowMap[row.Day] = row
	}
	day1 := "2026-03-01"
	day2 := "2026-03-02"
	if math.Abs(rowMap[day1].Revenue-0) > 0.000001 || math.Abs(rowMap[day1].Cost-30) > 0.000001 {
		t.Fatalf("unexpected day1 row: %+v", rowMap[day1])
	}
	if math.Abs(rowMap[day2].Revenue-60) > 0.000001 || math.Abs(rowMap[day2].Cost-40) > 0.000001 {
		t.Fatalf("unexpected day2 row: %+v", rowMap[day2])
	}
}

func TestGetProfitOverviewDeductsInWindowRefundForOutOfWindowOrder(t *testing.T) {
	repo, db := setupDashboardRepositoryTest(t)
	now := time.Now().UTC().Truncate(time.Second)
	startAt := now.AddDate(0, 0, -7)
	endAt := now.Add(time.Hour)

	category := createDashboardCategory(t, db, "dashboard-profit-period-refund-category")
	product := &models.Product{
		CategoryID:      category.ID,
		Slug:            "dashboard-profit-period-refund-product",
		TitleJSON:       models.JSON{"zh-CN": "????????"},
		PriceAmount:     models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
		PurchaseType:    constants.ProductPurchaseMember,
		FulfillmentType: constants.FulfillmentTypeManual,
		IsActive:        true,
	}
	if err := db.Create(product).Error; err != nil {
		t.Fatalf("create product failed: %v", err)
	}

	outsideOrder := &models.Order{
		OrderNo:        "DJ-PROFIT-OUTSIDE-ORDER",
		UserID:         1,
		Status:         constants.OrderStatusRefunded,
		Currency:       "CNY",
		OriginalAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
		DiscountAmount: models.NewMoneyFromDecimal(decimal.Zero),
		TotalAmount:    models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
		CreatedAt:      startAt.Add(-24 * time.Hour),
		UpdatedAt:      startAt.Add(-24 * time.Hour),
	}
	if err := db.Create(outsideOrder).Error; err != nil {
		t.Fatalf("create outside order failed: %v", err)
	}
	if err := db.Create(&models.OrderItem{
		OrderID:         outsideOrder.ID,
		ProductID:       product.ID,
		TitleJSON:       models.JSON{"zh-CN": "????????"},
		UnitPrice:       models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
		CostPrice:       models.NewMoneyFromDecimal(decimal.NewFromInt(40)),
		Quantity:        1,
		TotalPrice:      models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
		CouponDiscount:  models.NewMoneyFromDecimal(decimal.Zero),
		FulfillmentType: constants.FulfillmentTypeManual,
		CreatedAt:       startAt.Add(-24 * time.Hour),
		UpdatedAt:       startAt.Add(-24 * time.Hour),
	}).Error; err != nil {
		t.Fatalf("create outside order item failed: %v", err)
	}

	inWindowOrder := &models.Order{
		OrderNo:        "DJ-PROFIT-IN-WINDOW",
		UserID:         1,
		Status:         constants.OrderStatusCompleted,
		Currency:       "CNY",
		OriginalAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(60)),
		DiscountAmount: models.NewMoneyFromDecimal(decimal.Zero),
		TotalAmount:    models.NewMoneyFromDecimal(decimal.NewFromInt(60)),
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := db.Create(inWindowOrder).Error; err != nil {
		t.Fatalf("create in-window order failed: %v", err)
	}
	if err := db.Create(&models.OrderItem{
		OrderID:         inWindowOrder.ID,
		ProductID:       product.ID,
		TitleJSON:       models.JSON{"zh-CN": "????????"},
		UnitPrice:       models.NewMoneyFromDecimal(decimal.NewFromInt(60)),
		CostPrice:       models.NewMoneyFromDecimal(decimal.NewFromInt(20)),
		Quantity:        1,
		TotalPrice:      models.NewMoneyFromDecimal(decimal.NewFromInt(60)),
		CouponDiscount:  models.NewMoneyFromDecimal(decimal.Zero),
		FulfillmentType: constants.FulfillmentTypeManual,
		CreatedAt:       now,
		UpdatedAt:       now,
	}).Error; err != nil {
		t.Fatalf("create in-window order item failed: %v", err)
	}

	if err := db.Create(&models.OrderRefundRecord{
		UserID:    1,
		OrderID:   outsideOrder.ID,
		Type:      constants.OrderRefundTypeManual,
		Amount:    models.NewMoneyFromDecimal(decimal.NewFromInt(50)),
		Currency:  "CNY",
		CreatedAt: now,
		UpdatedAt: now,
	}).Error; err != nil {
		t.Fatalf("create refund record failed: %v", err)
	}

	result, err := repo.GetProfitOverview(startAt, endAt)
	if err != nil {
		t.Fatalf("get profit overview failed: %v", err)
	}
	if math.Abs(result.TotalRevenue-10) > 0.000001 {
		t.Fatalf("total revenue want 10 got %.2f", result.TotalRevenue)
	}
	if math.Abs(result.TotalCost-20) > 0.000001 {
		t.Fatalf("total cost want 20 got %.2f", result.TotalCost)
	}
}

func TestGetProfitTrendsIncludesRefundOnlyDayInWindow(t *testing.T) {
	repo, db := setupDashboardRepositoryTest(t)
	startAt := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	endAt := time.Date(2026, 3, 4, 0, 0, 0, 0, time.UTC)
	day1 := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	day2 := time.Date(2026, 3, 2, 11, 0, 0, 0, time.UTC)

	category := createDashboardCategory(t, db, "dashboard-profit-refund-only-day-category")
	product := &models.Product{
		CategoryID:      category.ID,
		Slug:            "dashboard-profit-refund-only-day-product",
		TitleJSON:       models.JSON{"zh-CN": "????????"},
		PriceAmount:     models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
		PurchaseType:    constants.ProductPurchaseMember,
		FulfillmentType: constants.FulfillmentTypeManual,
		IsActive:        true,
	}
	if err := db.Create(product).Error; err != nil {
		t.Fatalf("create product failed: %v", err)
	}

	inWindowOrder := &models.Order{
		OrderNo:        "DJ-PROFIT-TREND-IN-WINDOW",
		UserID:         1,
		Status:         constants.OrderStatusCompleted,
		Currency:       "CNY",
		OriginalAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(80)),
		DiscountAmount: models.NewMoneyFromDecimal(decimal.Zero),
		TotalAmount:    models.NewMoneyFromDecimal(decimal.NewFromInt(80)),
		CreatedAt:      day1,
		UpdatedAt:      day1,
	}
	if err := db.Create(inWindowOrder).Error; err != nil {
		t.Fatalf("create in-window order failed: %v", err)
	}
	if err := db.Create(&models.OrderItem{
		OrderID:         inWindowOrder.ID,
		ProductID:       product.ID,
		TitleJSON:       models.JSON{"zh-CN": "????????"},
		UnitPrice:       models.NewMoneyFromDecimal(decimal.NewFromInt(80)),
		CostPrice:       models.NewMoneyFromDecimal(decimal.NewFromInt(30)),
		Quantity:        1,
		TotalPrice:      models.NewMoneyFromDecimal(decimal.NewFromInt(80)),
		CouponDiscount:  models.NewMoneyFromDecimal(decimal.Zero),
		FulfillmentType: constants.FulfillmentTypeManual,
		CreatedAt:       day1,
		UpdatedAt:       day1,
	}).Error; err != nil {
		t.Fatalf("create in-window order item failed: %v", err)
	}

	outsideOrder := &models.Order{
		OrderNo:        "DJ-PROFIT-TREND-OUTSIDE-ORDER",
		UserID:         1,
		Status:         constants.OrderStatusRefunded,
		Currency:       "CNY",
		OriginalAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
		DiscountAmount: models.NewMoneyFromDecimal(decimal.Zero),
		TotalAmount:    models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
		CreatedAt:      startAt.Add(-48 * time.Hour),
		UpdatedAt:      startAt.Add(-48 * time.Hour),
	}
	if err := db.Create(outsideOrder).Error; err != nil {
		t.Fatalf("create outside order failed: %v", err)
	}
	if err := db.Create(&models.OrderItem{
		OrderID:         outsideOrder.ID,
		ProductID:       product.ID,
		TitleJSON:       models.JSON{"zh-CN": "????????"},
		UnitPrice:       models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
		CostPrice:       models.NewMoneyFromDecimal(decimal.NewFromInt(40)),
		Quantity:        1,
		TotalPrice:      models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
		CouponDiscount:  models.NewMoneyFromDecimal(decimal.Zero),
		FulfillmentType: constants.FulfillmentTypeManual,
		CreatedAt:       startAt.Add(-48 * time.Hour),
		UpdatedAt:       startAt.Add(-48 * time.Hour),
	}).Error; err != nil {
		t.Fatalf("create outside order item failed: %v", err)
	}

	if err := db.Create(&models.OrderRefundRecord{
		UserID:    1,
		OrderID:   outsideOrder.ID,
		Type:      constants.OrderRefundTypeManual,
		Amount:    models.NewMoneyFromDecimal(decimal.NewFromInt(30)),
		Currency:  "CNY",
		CreatedAt: day2,
		UpdatedAt: day2,
	}).Error; err != nil {
		t.Fatalf("create refund record failed: %v", err)
	}

	rows, err := repo.GetProfitTrends(startAt, endAt)
	if err != nil {
		t.Fatalf("get profit trends failed: %v", err)
	}
	rowMap := make(map[string]DashboardProfitTrendRow, len(rows))
	for _, row := range rows {
		rowMap[row.Day] = row
	}
	if math.Abs(rowMap["2026-03-01"].Revenue-80) > 0.000001 || math.Abs(rowMap["2026-03-01"].Cost-30) > 0.000001 {
		t.Fatalf("unexpected 2026-03-01 row: %+v", rowMap["2026-03-01"])
	}
	if math.Abs(rowMap["2026-03-02"].Revenue-(-30)) > 0.000001 || math.Abs(rowMap["2026-03-02"].Cost-0) > 0.000001 {
		t.Fatalf("unexpected 2026-03-02 row: %+v", rowMap["2026-03-02"])
	}
}
