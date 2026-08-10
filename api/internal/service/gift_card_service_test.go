package service

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/repository"

	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func setupGiftCardServiceTest(t *testing.T) (*GiftCardService, *WalletService, *gorm.DB) {
	t.Helper()
	dsn := fmt.Sprintf("file:gift_card_service_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.Order{},
		&models.OrderItem{},
		&models.Fulfillment{},
		&models.WalletAccount{},
		&models.WalletTransaction{},
		&models.Setting{},
		&models.GiftCardBatch{},
		&models.GiftCard{},
	); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}
	models.DB = db

	userRepo := repository.NewUserRepository(db)
	settingRepo := repository.NewSettingRepository(db)
	settingSvc := NewSettingService(settingRepo)
	walletSvc := NewWalletService(repository.NewWalletRepository(db), repository.NewOrderRepository(db), userRepo, nil, settingSvc)
	giftSvc := NewGiftCardService(repository.NewGiftCardRepository(db), userRepo, walletSvc, settingSvc)
	return giftSvc, walletSvc, db
}

func seedGiftCardUser(t *testing.T, db *gorm.DB, id uint) {
	t.Helper()
	user := models.User{
		ID:           id,
		Email:        fmt.Sprintf("gift_card_user_%d@example.com", id),
		PasswordHash: "hash",
		Status:       constants.UserStatusActive,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}
}

func TestGiftCardServiceGenerateGiftCards(t *testing.T) {
	svc, _, db := setupGiftCardServiceTest(t)
	adminID := uint(999)
	batch, created, err := svc.GenerateGiftCards(GenerateGiftCardsInput{
		Name:      "?????",
		Quantity:  3,
		Amount:    models.NewMoneyFromDecimal(decimal.RequireFromString("25.00")),
		CreatedBy: &adminID,
	})
	if err != nil {
		t.Fatalf("generate gift cards failed: %v", err)
	}
	if batch == nil || batch.ID == 0 {
		t.Fatalf("invalid batch result: %+v", batch)
	}
	if created != 3 {
		t.Fatalf("expected created=3, got: %d", created)
	}

	var count int64
	if err := db.Model(&models.GiftCard{}).Where("batch_id = ?", batch.ID).Count(&count).Error; err != nil {
		t.Fatalf("count gift cards failed: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 gift cards in batch, got: %d", count)
	}
}

func TestGiftCardServiceGenerateGiftCardsUsesSiteCurrency(t *testing.T) {
	svc, _, db := setupGiftCardServiceTest(t)
	settingRepo := repository.NewSettingRepository(db)
	settingSvc := NewSettingService(settingRepo)
	_, err := settingSvc.Update(constants.SettingKeySiteConfig, map[string]interface{}{
		constants.SettingFieldSiteCurrency: "USD",
	})
	if err != nil {
		t.Fatalf("set site currency failed: %v", err)
	}

	batch, created, err := svc.GenerateGiftCards(GenerateGiftCardsInput{
		Name:     "???????",
		Quantity: 2,
		Amount:   models.NewMoneyFromDecimal(decimal.RequireFromString("9.90")),
	})
	if err != nil {
		t.Fatalf("generate gift cards failed: %v", err)
	}
	if created != 2 {
		t.Fatalf("expected created=2, got: %d", created)
	}
	if batch == nil {
		t.Fatal("batch should not be nil")
	}
	if batch.Currency != "USD" {
		t.Fatalf("expected batch currency USD, got: %s", batch.Currency)
	}

	var cards []models.GiftCard
	if err := db.Where("batch_id = ?", batch.ID).Find(&cards).Error; err != nil {
		t.Fatalf("query gift cards failed: %v", err)
	}
	if len(cards) != 2 {
		t.Fatalf("expected 2 gift cards, got: %d", len(cards))
	}
	for _, card := range cards {
		if card.Currency != "USD" {
			t.Fatalf("expected card currency USD, got: %s", card.Currency)
		}
	}
}

func TestGiftCardServiceRedeemGiftCard(t *testing.T) {
	svc, walletSvc, db := setupGiftCardServiceTest(t)
	userID := uint(2001)
	seedGiftCardUser(t, db, userID)

	batch, _, err := svc.GenerateGiftCards(GenerateGiftCardsInput{
		Name:     "?????",
		Quantity: 1,
		Amount:   models.NewMoneyFromDecimal(decimal.RequireFromString("59.90")),
	})
	if err != nil {
		t.Fatalf("generate gift card failed: %v", err)
	}

	var card models.GiftCard
	if err := db.Where("batch_id = ?", batch.ID).First(&card).Error; err != nil {
		t.Fatalf("query generated card failed: %v", err)
	}

	redeemedCard, account, txn, err := svc.RedeemGiftCard(GiftCardRedeemInput{
		UserID: userID,
		Code:   card.Code,
	})
	if err != nil {
		t.Fatalf("redeem gift card failed: %v", err)
	}
	if redeemedCard == nil || redeemedCard.Status != models.GiftCardStatusRedeemed {
		t.Fatalf("unexpected redeemed card: %+v", redeemedCard)
	}
	if account == nil || !account.Balance.Decimal.Equal(decimal.RequireFromString("59.90")) {
		t.Fatalf("unexpected wallet account: %+v", account)
	}
	if txn == nil || txn.Type != constants.WalletTxnTypeGiftCard {
		t.Fatalf("unexpected wallet transaction: %+v", txn)
	}

	_, _, _, err = svc.RedeemGiftCard(GiftCardRedeemInput{
		UserID: userID,
		Code:   card.Code,
	})
	if !errors.Is(err, ErrGiftCardRedeemed) {
		t.Fatalf("expected ErrGiftCardRedeemed, got: %v", err)
	}

	accountAfter, err := walletSvc.GetAccount(userID)
	if err != nil {
		t.Fatalf("get account failed: %v", err)
	}
	if !accountAfter.Balance.Decimal.Equal(decimal.RequireFromString("59.90")) {
		t.Fatalf("unexpected account balance after duplicate redeem: %s", accountAfter.Balance.String())
	}
}

func TestGiftCardServiceRedeemExpiredGiftCard(t *testing.T) {
	svc, _, db := setupGiftCardServiceTest(t)
	userID := uint(2002)
	seedGiftCardUser(t, db, userID)
	expiredAt := time.Now().Add(-1 * time.Hour)

	card := models.GiftCard{
		Name:      "?????",
		Code:      "GC-EXPIRED-001",
		Amount:    models.NewMoneyFromDecimal(decimal.RequireFromString("10.00")),
		Currency:  "CNY",
		Status:    models.GiftCardStatusActive,
		ExpiresAt: &expiredAt,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := db.Create(&card).Error; err != nil {
		t.Fatalf("create expired gift card failed: %v", err)
	}

	_, _, _, err := svc.RedeemGiftCard(GiftCardRedeemInput{
		UserID: userID,
		Code:   card.Code,
	})
	if !errors.Is(err, ErrGiftCardExpired) {
		t.Fatalf("expected ErrGiftCardExpired, got: %v", err)
	}
}

func TestGiftCardServiceBatchUpdateStatusSkipsRedeemed(t *testing.T) {
	svc, _, db := setupGiftCardServiceTest(t)
	now := time.Now()
	userID := uint(2003)
	seedGiftCardUser(t, db, userID)

	activeCard := models.GiftCard{
		Name:      "??????",
		Code:      "GC-BATCH-ACTIVE-001",
		Amount:    models.NewMoneyFromDecimal(decimal.RequireFromString("20.00")),
		Currency:  "CNY",
		Status:    models.GiftCardStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := db.Create(&activeCard).Error; err != nil {
		t.Fatalf("create active card failed: %v", err)
	}

	redeemedAt := now.Add(-10 * time.Minute)
	redeemedCard := models.GiftCard{
		Name:           "??????",
		Code:           "GC-BATCH-REDEEMED-001",
		Amount:         models.NewMoneyFromDecimal(decimal.RequireFromString("30.00")),
		Currency:       "CNY",
		Status:         models.GiftCardStatusRedeemed,
		RedeemedAt:     &redeemedAt,
		RedeemedUserID: &userID,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := db.Create(&redeemedCard).Error; err != nil {
		t.Fatalf("create redeemed card failed: %v", err)
	}

	affected, err := svc.BatchUpdateStatus([]uint{activeCard.ID, redeemedCard.ID}, models.GiftCardStatusDisabled)
	if err != nil {
		t.Fatalf("batch update status failed: %v", err)
	}
	if affected != 1 {
		t.Fatalf("expected affected=1, got: %d", affected)
	}

	var checkActive models.GiftCard
	if err := db.First(&checkActive, activeCard.ID).Error; err != nil {
		t.Fatalf("query active card failed: %v", err)
	}
	if checkActive.Status != models.GiftCardStatusDisabled {
		t.Fatalf("expected active card status disabled, got: %s", checkActive.Status)
	}

	var checkRedeemed models.GiftCard
	if err := db.First(&checkRedeemed, redeemedCard.ID).Error; err != nil {
		t.Fatalf("query redeemed card failed: %v", err)
	}
	if checkRedeemed.Status != models.GiftCardStatusRedeemed {
		t.Fatalf("expected redeemed card status unchanged, got: %s", checkRedeemed.Status)
	}
}
