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

func setupWalletRepositoryTest(t *testing.T) (*GormWalletRepository, *gorm.DB) {
	t.Helper()
	dsn := fmt.Sprintf("file:wallet_repo_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.WalletRechargeOrder{},
	); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}
	return NewWalletRepository(db), db
}

func TestWalletRepositoryListRechargeOrdersAdmin(t *testing.T) {
	repo, db := setupWalletRepositoryTest(t)
	now := time.Now().UTC().Truncate(time.Second)

	user1 := models.User{
		Email:        "alpha_wallet_repo@example.com",
		DisplayName:  "Alpha",
		PasswordHash: "hash",
		Status:       constants.UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	user2 := models.User{
		Email:        "beta_wallet_repo@example.com",
		DisplayName:  "Beta",
		PasswordHash: "hash",
		Status:       constants.UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := db.Create(&user1).Error; err != nil {
		t.Fatalf("create user1 failed: %v", err)
	}
	if err := db.Create(&user2).Error; err != nil {
		t.Fatalf("create user2 failed: %v", err)
	}

	paidAt1 := now.Add(-2 * time.Hour)
	paidAt3 := now.Add(-30 * time.Minute)
	orders := []models.WalletRechargeOrder{
		{
			RechargeNo:      "DJR-A001",
			UserID:          user1.ID,
			PaymentID:       1001,
			ChannelID:       11,
			ProviderType:    constants.PaymentProviderOfficial,
			ChannelType:     constants.PaymentChannelTypeAlipay,
			InteractionMode: constants.PaymentInteractionRedirect,
			Amount:          models.NewMoneyFromDecimal(decimal.RequireFromString("50.00")),
			PayableAmount:   models.NewMoneyFromDecimal(decimal.RequireFromString("50.00")),
			FeeRate:         models.NewMoneyFromDecimal(decimal.Zero),
			FeeAmount:       models.NewMoneyFromDecimal(decimal.Zero),
			Currency:        "CNY",
			Status:          constants.WalletRechargeStatusSuccess,
			PaidAt:          &paidAt1,
			CreatedAt:       now.Add(-3 * time.Hour),
			UpdatedAt:       now.Add(-2 * time.Hour),
		},
		{
			RechargeNo:      "DJR-A002",
			UserID:          user1.ID,
			PaymentID:       1002,
			ChannelID:       11,
			ProviderType:    constants.PaymentProviderOfficial,
			ChannelType:     constants.PaymentChannelTypeWechat,
			InteractionMode: constants.PaymentInteractionQR,
			Amount:          models.NewMoneyFromDecimal(decimal.RequireFromString("80.00")),
			PayableAmount:   models.NewMoneyFromDecimal(decimal.RequireFromString("80.00")),
			FeeRate:         models.NewMoneyFromDecimal(decimal.Zero),
			FeeAmount:       models.NewMoneyFromDecimal(decimal.Zero),
			Currency:        "CNY",
			Status:          constants.WalletRechargeStatusPending,
			CreatedAt:       now.Add(-20 * time.Minute),
			UpdatedAt:       now.Add(-20 * time.Minute),
		},
		{
			RechargeNo:      "DJR-B001",
			UserID:          user2.ID,
			PaymentID:       2001,
			ChannelID:       12,
			ProviderType:    constants.PaymentProviderOfficial,
			ChannelType:     constants.PaymentChannelTypeAlipay,
			InteractionMode: constants.PaymentInteractionRedirect,
			Amount:          models.NewMoneyFromDecimal(decimal.RequireFromString("120.00")),
			PayableAmount:   models.NewMoneyFromDecimal(decimal.RequireFromString("120.00")),
			FeeRate:         models.NewMoneyFromDecimal(decimal.Zero),
			FeeAmount:       models.NewMoneyFromDecimal(decimal.Zero),
			Currency:        "CNY",
			Status:          constants.WalletRechargeStatusSuccess,
			PaidAt:          &paidAt3,
			CreatedAt:       now.Add(-40 * time.Minute),
			UpdatedAt:       now.Add(-30 * time.Minute),
		},
	}
	if err := db.Create(&orders).Error; err != nil {
		t.Fatalf("create recharge orders failed: %v", err)
	}

	t.Run("filter by user keyword", func(t *testing.T) {
		rows, total, err := repo.ListRechargeOrdersAdmin(WalletRechargeListFilter{
			Page:        1,
			PageSize:    20,
			UserKeyword: "alpha_wallet_repo",
		})
		if err != nil {
			t.Fatalf("list by user keyword failed: %v", err)
		}
		if total != 2 {
			t.Fatalf("total want 2 got %d", total)
		}
		if len(rows) != 2 {
			t.Fatalf("rows len want 2 got %d", len(rows))
		}
		for _, row := range rows {
			if row.UserID != user1.ID {
				t.Fatalf("expected only user1 rows, got user_id=%d", row.UserID)
			}
		}
	})

	t.Run("filter by status and paid range", func(t *testing.T) {
		from := now.Add(-3 * time.Hour)
		to := now.Add(-90 * time.Minute)
		rows, total, err := repo.ListRechargeOrdersAdmin(WalletRechargeListFilter{
			Page:      1,
			PageSize:  20,
			Status:    constants.WalletRechargeStatusSuccess,
			PaidFrom:  &from,
			PaidTo:    &to,
			ChannelID: 11,
		})
		if err != nil {
			t.Fatalf("list by status/paid range failed: %v", err)
		}
		if total != 1 {
			t.Fatalf("total want 1 got %d", total)
		}
		if len(rows) != 1 {
			t.Fatalf("rows len want 1 got %d", len(rows))
		}
		if rows[0].RechargeNo != "DJR-A001" {
			t.Fatalf("unexpected recharge_no=%s", rows[0].RechargeNo)
		}
	})
}
