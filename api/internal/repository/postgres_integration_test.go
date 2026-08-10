//go:build integration
// +build integration

package repository

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"

	"github.com/shopspring/decimal"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// setupPostgresIntegrationDB ??? PostgreSQL ????????
func setupPostgresIntegrationDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := strings.TrimSpace(os.Getenv("TEST_POSTGRES_DSN"))
	if dsn == "" {
		t.Skip("skip postgres integration test: TEST_POSTGRES_DSN is empty")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open postgres failed: %v", err)
	}

	cleanupModels := []interface{}{
		&models.OrderItem{},
		&models.Payment{},
		&models.Order{},
		&models.Product{},
		&models.Category{},
		&models.Banner{},
		&models.Post{},
	}
	_ = db.Migrator().DropTable(cleanupModels...)

	if err := db.AutoMigrate(
		&models.Category{},
		&models.Product{},
		&models.Post{},
		&models.Banner{},
		&models.Order{},
		&models.OrderItem{},
		&models.Payment{},
	); err != nil {
		t.Fatalf("migrate postgres models failed: %v", err)
	}

	t.Cleanup(func() {
		_ = db.Migrator().DropTable(cleanupModels...)
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}

func TestPostgresLocalizedJSONSearchRepositories(t *testing.T) {
	db := setupPostgresIntegrationDB(t)

	category := &models.Category{
		Slug:     "pg-category",
		NameJSON: models.JSON{"zh-CN": "Postgres ??"},
	}
	if err := db.Create(category).Error; err != nil {
		t.Fatalf("create category failed: %v", err)
	}

	productRepo := NewProductRepository(db)
	product := &models.Product{
		CategoryID:       category.ID,
		Slug:             "pg-product-rocket",
		TitleJSON:        models.JSON{"zh-CN": "????"},
		DescriptionJSON:  models.JSON{"en-US": "rocket booster package"},
		PriceAmount:      models.NewMoneyFromDecimal(decimal.NewFromInt(99)),
		PurchaseType:     constants.ProductPurchaseMember,
		FulfillmentType:  constants.FulfillmentTypeManual,
		ManualStockTotal: 10,
		IsActive:         true,
	}
	if err := productRepo.Create(product); err != nil {
		t.Fatalf("create product failed: %v", err)
	}

	productRows, productTotal, err := productRepo.List(ProductListFilter{
		Page:   1,
		Search: "??",
	})
	if err != nil {
		t.Fatalf("product list search zh-CN failed: %v", err)
	}
	if productTotal != 1 || len(productRows) != 1 {
		t.Fatalf("product list search zh-CN want 1 got total=%d len=%d", productTotal, len(productRows))
	}

	productRows, productTotal, err = productRepo.List(ProductListFilter{
		Page:   1,
		Search: "booster",
	})
	if err != nil {
		t.Fatalf("product list search en-US failed: %v", err)
	}
	if productTotal != 1 || len(productRows) != 1 {
		t.Fatalf("product list search en-US want 1 got total=%d len=%d", productTotal, len(productRows))
	}

	postRepo := NewPostRepository(db)
	post := &models.Post{
		Slug:        "pg-post-release",
		Type:        "notice",
		TitleJSON:   models.JSON{"en-US": "Release Notes"},
		IsPublished: true,
	}
	if err := postRepo.Create(post); err != nil {
		t.Fatalf("create post failed: %v", err)
	}

	postRows, postTotal, err := postRepo.List(PostListFilter{
		Page:   1,
		Search: "Release",
	})
	if err != nil {
		t.Fatalf("post list search failed: %v", err)
	}
	if postTotal != 1 || len(postRows) != 1 {
		t.Fatalf("post list search want 1 got total=%d len=%d", postTotal, len(postRows))
	}

	bannerRepo := NewBannerRepository(db)
	banner := &models.Banner{
		Name:      "pg-home-banner",
		Position:  "home",
		TitleJSON: models.JSON{"zh-CN": "????"},
		Image:     "/banner.png",
		LinkType:  "none",
		IsActive:  true,
	}
	if err := bannerRepo.Create(banner); err != nil {
		t.Fatalf("create banner failed: %v", err)
	}

	bannerRows, bannerTotal, err := bannerRepo.List(BannerListFilter{
		Page:   1,
		Search: "??",
	})
	if err != nil {
		t.Fatalf("banner list search failed: %v", err)
	}
	if bannerTotal != 1 || len(bannerRows) != 1 {
		t.Fatalf("banner list search want 1 got total=%d len=%d", bannerTotal, len(bannerRows))
	}
}

func TestPostgresDashboardQueries(t *testing.T) {
	db := setupPostgresIntegrationDB(t)
	repo := NewDashboardRepository(db)
	now := time.Now().UTC().Truncate(time.Second)

	category := &models.Category{
		Slug:     "pg-dashboard-category",
		NameJSON: models.JSON{"zh-CN": "?????"},
	}
	if err := db.Create(category).Error; err != nil {
		t.Fatalf("create category failed: %v", err)
	}

	product := &models.Product{
		CategoryID:      category.ID,
		Slug:            "pg-dashboard-product",
		TitleJSON:       models.JSON{"zh-CN": "?????"},
		PriceAmount:     models.NewMoneyFromDecimal(decimal.NewFromInt(120)),
		PurchaseType:    constants.ProductPurchaseMember,
		FulfillmentType: constants.FulfillmentTypeManual,
		IsActive:        true,
	}
	if err := db.Create(product).Error; err != nil {
		t.Fatalf("create product failed: %v", err)
	}

	order := &models.Order{
		OrderNo:        "PG-ORDER-001",
		UserID:         1,
		Status:         constants.OrderStatusPaid,
		Currency:       "USD",
		OriginalAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(120)),
		DiscountAmount: models.NewMoneyFromDecimal(decimal.Zero),
		TotalAmount:    models.NewMoneyFromDecimal(decimal.NewFromInt(120)),
		CreatedAt:      now,
	}
	if err := db.Create(order).Error; err != nil {
		t.Fatalf("create order failed: %v", err)
	}

	orderItem := &models.OrderItem{
		OrderID:           order.ID,
		ProductID:         product.ID,
		TitleJSON:         models.JSON{"zh-CN": "?????"},
		UnitPrice:         models.NewMoneyFromDecimal(decimal.NewFromInt(120)),
		Quantity:          2,
		TotalPrice:        models.NewMoneyFromDecimal(decimal.NewFromInt(240)),
		CouponDiscount:    models.NewMoneyFromDecimal(decimal.NewFromInt(20)),
		PromotionDiscount: models.NewMoneyFromDecimal(decimal.NewFromInt(10)),
		FulfillmentType:   constants.FulfillmentTypeManual,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := db.Create(orderItem).Error; err != nil {
		t.Fatalf("create order item failed: %v", err)
	}

	payment := &models.Payment{
		OrderID:         order.ID,
		ChannelID:       1,
		ProviderType:    constants.PaymentProviderOfficial,
		ChannelType:     constants.PaymentChannelTypeAlipay,
		InteractionMode: constants.PaymentInteractionRedirect,
		Amount:          models.NewMoneyFromDecimal(decimal.NewFromInt(120)),
		FeeRate:         models.NewMoneyFromDecimal(decimal.Zero),
		FeeAmount:       models.NewMoneyFromDecimal(decimal.Zero),
		Currency:        "USD",
		Status:          constants.PaymentStatusSuccess,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := db.Create(payment).Error; err != nil {
		t.Fatalf("create payment failed: %v", err)
	}

	startAt := now.Add(-time.Hour)
	endAt := now.Add(time.Hour)

	topProducts, err := repo.GetTopProducts(startAt, endAt, 5)
	if err != nil {
		t.Fatalf("get top products failed: %v", err)
	}
	if len(topProducts) != 1 {
		t.Fatalf("top products len want 1 got %d", len(topProducts))
	}
	if topProducts[0].Title != "?????" {
		t.Fatalf("top product title want ????? got %s", topProducts[0].Title)
	}

	orderTrends, err := repo.GetOrderTrends(startAt, endAt)
	if err != nil {
		t.Fatalf("get order trends failed: %v", err)
	}
	if len(orderTrends) == 0 {
		t.Fatalf("order trends should not be empty")
	}
	if strings.TrimSpace(orderTrends[0].Day) == "" {
		t.Fatalf("order trend day should not be empty")
	}

	paymentTrends, err := repo.GetPaymentTrends(startAt, endAt)
	if err != nil {
		t.Fatalf("get payment trends failed: %v", err)
	}
	if len(paymentTrends) == 0 {
		t.Fatalf("payment trends should not be empty")
	}
	if strings.TrimSpace(paymentTrends[0].Day) == "" {
		t.Fatalf("payment trend day should not be empty")
	}
}
