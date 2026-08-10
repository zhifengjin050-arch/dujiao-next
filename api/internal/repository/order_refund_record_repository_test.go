package repository

import (
	"fmt"
	"testing"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"

	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func setupOrderRefundRecordRepositoryTest(t *testing.T) (*GormOrderRefundRecordRepository, *gorm.DB) {
	t.Helper()
	dsn := fmt.Sprintf("file:order_refund_record_repo_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.UserOAuthIdentity{},
		&models.Order{},
		&models.OrderItem{},
		&models.OrderRefundRecord{},
	); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}
	return NewOrderRefundRecordRepository(db), db
}

func TestOrderRefundRecordRepositoryListAdminFilters(t *testing.T) {
	repo, db := setupOrderRefundRecordRepositoryTest(t)
	now := time.Now().UTC().Truncate(time.Second)

	user1 := &models.User{
		Email:        "member-alpha@example.com",
		DisplayName:  "Alpha",
		PasswordHash: "hash",
		Status:       constants.UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	user2 := &models.User{
		Email:        "member-beta@example.com",
		DisplayName:  "Beta",
		PasswordHash: "hash",
		Status:       constants.UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := db.Create(user1).Error; err != nil {
		t.Fatalf("create user1 failed: %v", err)
	}
	if err := db.Create(user2).Error; err != nil {
		t.Fatalf("create user2 failed: %v", err)
	}
	if err := db.Create(&models.UserOAuthIdentity{
		UserID:         user1.ID,
		Provider:       "telegram",
		ProviderUserID: "6059928735",
		Username:       "tg_alpha",
		CreatedAt:      now,
		UpdatedAt:      now,
	}).Error; err != nil {
		t.Fatalf("create user oauth identity failed: %v", err)
	}

	order1 := &models.Order{
		OrderNo:          "DJ-REFUND-001",
		UserID:           user1.ID,
		Status:           constants.OrderStatusCompleted,
		Currency:         "CNY",
		OriginalAmount:   models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
		DiscountAmount:   models.NewMoneyFromDecimal(decimal.Zero),
		TotalAmount:      models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
		WalletPaidAmount: models.NewMoneyFromDecimal(decimal.Zero),
		OnlinePaidAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
		RefundedAmount:   models.NewMoneyFromDecimal(decimal.Zero),
		CreatedAt:        now.Add(-4 * time.Hour),
		UpdatedAt:        now.Add(-4 * time.Hour),
	}
	order2 := &models.Order{
		OrderNo:          "DJ-REFUND-002",
		UserID:           0,
		GuestEmail:       "guest-refund@example.com",
		Status:           constants.OrderStatusCompleted,
		Currency:         "CNY",
		OriginalAmount:   models.NewMoneyFromDecimal(decimal.NewFromInt(80)),
		DiscountAmount:   models.NewMoneyFromDecimal(decimal.Zero),
		TotalAmount:      models.NewMoneyFromDecimal(decimal.NewFromInt(80)),
		WalletPaidAmount: models.NewMoneyFromDecimal(decimal.Zero),
		OnlinePaidAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(80)),
		RefundedAmount:   models.NewMoneyFromDecimal(decimal.Zero),
		CreatedAt:        now.Add(-3 * time.Hour),
		UpdatedAt:        now.Add(-3 * time.Hour),
	}
	order3 := &models.Order{
		OrderNo:          "DJ-REFUND-003",
		UserID:           user2.ID,
		Status:           constants.OrderStatusCompleted,
		Currency:         "CNY",
		OriginalAmount:   models.NewMoneyFromDecimal(decimal.NewFromInt(60)),
		DiscountAmount:   models.NewMoneyFromDecimal(decimal.Zero),
		TotalAmount:      models.NewMoneyFromDecimal(decimal.NewFromInt(60)),
		WalletPaidAmount: models.NewMoneyFromDecimal(decimal.Zero),
		OnlinePaidAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(60)),
		RefundedAmount:   models.NewMoneyFromDecimal(decimal.Zero),
		CreatedAt:        now.Add(-2 * time.Hour),
		UpdatedAt:        now.Add(-2 * time.Hour),
	}
	for _, order := range []*models.Order{order1, order2, order3} {
		if err := db.Create(order).Error; err != nil {
			t.Fatalf("create order failed: %v", err)
		}
	}

	items := []models.OrderItem{
		{
			OrderID:         order1.ID,
			ProductID:       1,
			TitleJSON:       models.JSON{"zh-CN": "????A"},
			UnitPrice:       models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
			CostPrice:       models.NewMoneyFromDecimal(decimal.NewFromInt(50)),
			Quantity:        1,
			TotalPrice:      models.NewMoneyFromDecimal(decimal.NewFromInt(100)),
			CouponDiscount:  models.NewMoneyFromDecimal(decimal.Zero),
			FulfillmentType: constants.FulfillmentTypeManual,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		{
			OrderID:         order2.ID,
			ProductID:       2,
			TitleJSON:       models.JSON{"zh-CN": "????B"},
			UnitPrice:       models.NewMoneyFromDecimal(decimal.NewFromInt(80)),
			CostPrice:       models.NewMoneyFromDecimal(decimal.NewFromInt(40)),
			Quantity:        1,
			TotalPrice:      models.NewMoneyFromDecimal(decimal.NewFromInt(80)),
			CouponDiscount:  models.NewMoneyFromDecimal(decimal.Zero),
			FulfillmentType: constants.FulfillmentTypeManual,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		{
			OrderID:         order3.ID,
			ProductID:       3,
			TitleJSON:       models.JSON{"zh-CN": "????C"},
			UnitPrice:       models.NewMoneyFromDecimal(decimal.NewFromInt(60)),
			CostPrice:       models.NewMoneyFromDecimal(decimal.NewFromInt(30)),
			Quantity:        1,
			TotalPrice:      models.NewMoneyFromDecimal(decimal.NewFromInt(60)),
			CouponDiscount:  models.NewMoneyFromDecimal(decimal.Zero),
			FulfillmentType: constants.FulfillmentTypeManual,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
	}
	for idx := range items {
		if err := db.Create(&items[idx]).Error; err != nil {
			t.Fatalf("create order item failed: %v", err)
		}
	}

	record1 := &models.OrderRefundRecord{
		UserID:     user1.ID,
		OrderID:    order1.ID,
		Type:       constants.OrderRefundTypeManual,
		Amount:     models.NewMoneyFromDecimal(decimal.NewFromInt(10)),
		Currency:   "CNY",
		Remark:     "member refund",
		CreatedAt:  now.Add(-90 * time.Minute),
		UpdatedAt:  now.Add(-90 * time.Minute),
		GuestEmail: "",
	}
	record2 := &models.OrderRefundRecord{
		UserID:     0,
		GuestEmail: "guest-refund@example.com",
		OrderID:    order2.ID,
		Type:       constants.OrderRefundTypeWallet,
		Amount:     models.NewMoneyFromDecimal(decimal.NewFromInt(20)),
		Currency:   "CNY",
		Remark:     "guest refund",
		CreatedAt:  now.Add(-30 * time.Minute),
		UpdatedAt:  now.Add(-30 * time.Minute),
	}
	record3 := &models.OrderRefundRecord{
		UserID:     user2.ID,
		OrderID:    order3.ID,
		Type:       constants.OrderRefundTypeManual,
		Amount:     models.NewMoneyFromDecimal(decimal.NewFromInt(5)),
		Currency:   "CNY",
		Remark:     "old refund",
		CreatedAt:  now.Add(-10 * 24 * time.Hour),
		UpdatedAt:  now.Add(-10 * 24 * time.Hour),
		GuestEmail: "",
	}
	for _, record := range []*models.OrderRefundRecord{record1, record2, record3} {
		if err := db.Create(record).Error; err != nil {
			t.Fatalf("create refund record failed: %v", err)
		}
	}

	rows, total, err := repo.ListAdmin(OrderRefundRecordListFilter{
		Page:        1,
		PageSize:    20,
		UserKeyword: "tg_alpha",
	})
	if err != nil {
		t.Fatalf("list by user keyword failed: %v", err)
	}
	if total != 1 || len(rows) != 1 || rows[0].ID != record1.ID {
		t.Fatalf("unexpected list by user keyword result: total=%d rows=%+v", total, rows)
	}

	rows, total, err = repo.ListAdmin(OrderRefundRecordListFilter{
		Page:           1,
		PageSize:       20,
		OrderNo:        "DJ-REFUND-002",
		GuestEmail:     "guest-refund@example.com",
		ProductKeyword: "????",
	})
	if err != nil {
		t.Fatalf("list by order/guest/product filter failed: %v", err)
	}
	if total != 1 || len(rows) != 1 || rows[0].ID != record2.ID {
		t.Fatalf("unexpected list by order/guest/product result: total=%d rows=%+v", total, rows)
	}

	createdFrom := now.Add(-40 * time.Minute)
	createdTo := now.Add(10 * time.Minute)
	rows, total, err = repo.ListAdmin(OrderRefundRecordListFilter{
		Page:        1,
		PageSize:    20,
		CreatedFrom: &createdFrom,
		CreatedTo:   &createdTo,
	})
	if err != nil {
		t.Fatalf("list by created range failed: %v", err)
	}
	if total != 1 || len(rows) != 1 || rows[0].ID != record2.ID {
		t.Fatalf("unexpected list by created range result: total=%d rows=%+v", total, rows)
	}
}

func TestOrderRefundRecordRepositoryGetByID(t *testing.T) {
	repo, db := setupOrderRefundRecordRepositoryTest(t)
	now := time.Now().UTC().Truncate(time.Second)
	record := &models.OrderRefundRecord{
		UserID:     1,
		OrderID:    2,
		Type:       constants.OrderRefundTypeManual,
		Amount:     models.NewMoneyFromDecimal(decimal.NewFromInt(12)),
		Currency:   "CNY",
		CreatedAt:  now,
		UpdatedAt:  now,
		GuestEmail: "",
	}
	if err := db.Create(record).Error; err != nil {
		t.Fatalf("create refund record failed: %v", err)
	}

	got, err := repo.GetByID(record.ID)
	if err != nil {
		t.Fatalf("get by id failed: %v", err)
	}
	if got == nil || got.ID != record.ID {
		t.Fatalf("unexpected get by id result: %+v", got)
	}

	missing, err := repo.GetByID(record.ID + 100)
	if err != nil {
		t.Fatalf("get missing by id failed: %v", err)
	}
	if missing != nil {
		t.Fatalf("missing record should be nil, got %+v", missing)
	}
}
