package channel

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/dujiao-next/internal/config"
	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/provider"
	"github.com/dujiao-next/internal/repository"
	"github.com/dujiao-next/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type channelGiftCardTestResponse struct {
	StatusCode int                    `json:"status_code"`
	Msg        string                 `json:"msg"`
	Data       map[string]interface{} `json:"data"`
	ErrorCode  string                 `json:"error_code"`
}

func setupChannelGiftCardHandlerTest(t *testing.T) (*gorm.DB, *httptest.Server) {
	t.Helper()

	dsn := fmt.Sprintf("file:channel_wallet_gift_card_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(
		&models.User{},
		&models.UserOAuthIdentity{},
		&models.EmailVerifyCode{},
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
	identityRepo := repository.NewUserOAuthIdentityRepository(db)
	emailVerifyRepo := repository.NewEmailVerifyCodeRepository(db)
	orderRepo := repository.NewOrderRepository(db)
	walletRepo := repository.NewWalletRepository(db)
	settingRepo := repository.NewSettingRepository(db)
	giftCardRepo := repository.NewGiftCardRepository(db)

	settingSvc := service.NewSettingService(settingRepo)
	walletSvc := service.NewWalletService(walletRepo, orderRepo, userRepo, nil)
	giftCardSvc := service.NewGiftCardService(giftCardRepo, userRepo, walletSvc, settingSvc)
	userAuthSvc := service.NewUserAuthService(&config.Config{}, userRepo, identityRepo, emailVerifyRepo, nil, nil, nil)

	handler := New(&provider.Container{
		UserRepo:              userRepo,
		UserOAuthIdentityRepo: identityRepo,
		EmailVerifyCodeRepo:   emailVerifyRepo,
		OrderRepo:             orderRepo,
		WalletRepo:            walletRepo,
		SettingRepo:           settingRepo,
		GiftCardRepo:          giftCardRepo,
		UserAuthService:       userAuthSvc,
		WalletService:         walletSvc,
		SettingService:        settingSvc,
		GiftCardService:       giftCardSvc,
	})

	ginRouter := newTestChannelRouter(handler)
	server := httptest.NewServer(ginRouter)
	t.Cleanup(server.Close)
	return db, server
}

func newTestChannelRouter(handler *Handler) http.Handler {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/channel/wallet/gift-card/redeem", handler.RedeemGiftCard)
	return router
}

func seedChannelGiftCard(t *testing.T, db *gorm.DB, card models.GiftCard) models.GiftCard {
	t.Helper()
	if err := db.Create(&card).Error; err != nil {
		t.Fatalf("create gift card failed: %v", err)
	}
	return card
}

func postChannelGiftCardRedeem(t *testing.T, server *httptest.Server, body map[string]interface{}) *http.Response {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request failed: %v", err)
	}
	resp, err := http.Post(server.URL+"/api/v1/channel/wallet/gift-card/redeem", "application/json", bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("post redeem gift card failed: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	return resp
}

func decodeChannelGiftCardResponse(t *testing.T, resp *http.Response) channelGiftCardTestResponse {
	t.Helper()
	var payload channelGiftCardTestResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode response failed: %v", err)
	}
	return payload
}

func TestRedeemGiftCardChannelHandlerSuccess(t *testing.T) {
	db, server := setupChannelGiftCardHandlerTest(t)
	card := seedChannelGiftCard(t, db, models.GiftCard{
		Name:      "Telegram ???",
		Code:      "GC-CHANNEL-SUCCESS-001",
		Amount:    models.NewMoneyFromDecimal(decimal.RequireFromString("88.80")),
		Currency:  "CNY",
		Status:    models.GiftCardStatusActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	resp := postChannelGiftCardRedeem(t, server, map[string]interface{}{
		"telegram_user_id": "998877",
		"code":             card.Code,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected http status 200, got %d", resp.StatusCode)
	}

	payload := decodeChannelGiftCardResponse(t, resp)
	if payload.StatusCode != 0 {
		t.Fatalf("expected status_code=0, got %d", payload.StatusCode)
	}
	if payload.Msg != "success" {
		t.Fatalf("expected msg=success, got %s", payload.Msg)
	}

	giftCardData, ok := payload.Data["gift_card"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected gift_card map, got %T", payload.Data["gift_card"])
	}
	if giftCardData["code"] != card.Code {
		t.Fatalf("expected gift_card.code=%s, got %v", card.Code, giftCardData["code"])
	}
	if giftCardData["status"] != models.GiftCardStatusRedeemed {
		t.Fatalf("expected gift_card.status=redeemed, got %v", giftCardData["status"])
	}

	walletData, ok := payload.Data["wallet"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected wallet map, got %T", payload.Data["wallet"])
	}
	if walletData["balance"] != "88.80" {
		t.Fatalf("expected wallet.balance=88.80, got %v", walletData["balance"])
	}

	txnData, ok := payload.Data["transaction"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected transaction map, got %T", payload.Data["transaction"])
	}
	if txnData["type"] != constants.WalletTxnTypeGiftCard {
		t.Fatalf("expected transaction.type=%s, got %v", constants.WalletTxnTypeGiftCard, txnData["type"])
	}
	if payload.Data["wallet_delta"] != "88.80" {
		t.Fatalf("expected wallet_delta=88.80, got %v", payload.Data["wallet_delta"])
	}

	var identity models.UserOAuthIdentity
	if err := db.Where("provider = ? AND provider_user_id = ?", constants.UserOAuthProviderTelegram, "998877").First(&identity).Error; err != nil {
		t.Fatalf("expected provisioned telegram identity: %v", err)
	}

	var account models.WalletAccount
	if err := db.Where("user_id = ?", identity.UserID).First(&account).Error; err != nil {
		t.Fatalf("expected wallet account: %v", err)
	}
	if account.Balance.String() != "88.80" {
		t.Fatalf("expected stored wallet balance=88.80, got %s", account.Balance.String())
	}

	var refreshedCard models.GiftCard
	if err := db.First(&refreshedCard, card.ID).Error; err != nil {
		t.Fatalf("reload gift card failed: %v", err)
	}
	if refreshedCard.Status != models.GiftCardStatusRedeemed {
		t.Fatalf("expected stored gift card status redeemed, got %s", refreshedCard.Status)
	}
	if refreshedCard.RedeemedUserID == nil || *refreshedCard.RedeemedUserID != identity.UserID {
		t.Fatalf("expected redeemed_user_id=%d, got %+v", identity.UserID, refreshedCard.RedeemedUserID)
	}
}

func TestRedeemGiftCardChannelHandlerReturnsMappedRedeemedError(t *testing.T) {
	db, server := setupChannelGiftCardHandlerTest(t)
	redeemedUserID := uint(321)
	redeemedAt := time.Now().Add(-10 * time.Minute)
	seedChannelGiftCard(t, db, models.GiftCard{
		Name:           "??????",
		Code:           "GC-CHANNEL-REDEEMED-001",
		Amount:         models.NewMoneyFromDecimal(decimal.RequireFromString("50.00")),
		Currency:       "CNY",
		Status:         models.GiftCardStatusRedeemed,
		RedeemedAt:     &redeemedAt,
		RedeemedUserID: &redeemedUserID,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	})

	resp := postChannelGiftCardRedeem(t, server, map[string]interface{}{
		"channel_user_id": "556677",
		"code":            "GC-CHANNEL-REDEEMED-001",
	})
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected http status 400, got %d", resp.StatusCode)
	}

	payload := decodeChannelGiftCardResponse(t, resp)
	if payload.StatusCode != 400 {
		t.Fatalf("expected status_code=400, got %d", payload.StatusCode)
	}
	if payload.ErrorCode != "gift_card_redeemed" {
		t.Fatalf("expected error_code=gift_card_redeemed, got %s", payload.ErrorCode)
	}

	var identity models.UserOAuthIdentity
	if err := db.Where("provider = ? AND provider_user_id = ?", constants.UserOAuthProviderTelegram, "556677").First(&identity).Error; err != nil {
		t.Fatalf("expected provisioned telegram identity on failure path: %v", err)
	}

	var walletCount int64
	if err := db.Model(&models.WalletAccount{}).Where("user_id = ?", identity.UserID).Count(&walletCount).Error; err != nil {
		t.Fatalf("count wallet accounts failed: %v", err)
	}
	if walletCount != 0 {
		t.Fatalf("expected no wallet account created on redeemed error, got %d", walletCount)
	}
}
