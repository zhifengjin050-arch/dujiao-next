package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/dujiao-next/internal/config"
	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/repository"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestLoginWithTelegramMiniAppCreatesUserIdentityAndToken(t *testing.T) {
	dsn := fmt.Sprintf("file:user_auth_service_miniapp_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.UserOAuthIdentity{}, &models.EmailVerifyCode{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	cfg := &config.Config{
		UserJWT: config.JWTConfig{
			SecretKey:   "user-jwt-test-secret",
			ExpireHours: 24,
		},
	}
	telegramSvc := NewTelegramAuthService(config.TelegramAuthConfig{
		Enabled:            true,
		BotToken:           "test-bot-token",
		LoginExpireSeconds: 300,
		ReplayTTLSeconds:   300,
	})
	telegramSvc.replaySetNX = func(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
		return true, nil
	}

	svc := NewUserAuthService(
		cfg,
		repository.NewUserRepository(db),
		repository.NewUserOAuthIdentityRepository(db),
		repository.NewEmailVerifyCodeRepository(db),
		nil,
		nil,
		telegramSvc,
	)

	initData := buildTestTelegramMiniAppInitData(t, "test-bot-token", time.Now().Unix(), `{"id":987654,"first_name":"Mini","last_name":"Buyer","username":"mini_buyer"}`)
	user, token, expiresAt, err := svc.LoginWithTelegramMiniApp(LoginWithTelegramMiniAppInput{
		InitData: initData,
		Context:  context.Background(),
	})
	if err != nil {
		t.Fatalf("LoginWithTelegramMiniApp returned error: %v", err)
	}
	if user == nil {
		t.Fatalf("expected user")
	}
	if user.Email != "telegram_987654@login.local" {
		t.Fatalf("user email mismatch: %s", user.Email)
	}
	if user.Status != constants.UserStatusActive {
		t.Fatalf("user status want active got %s", user.Status)
	}
	if token == "" {
		t.Fatalf("expected token")
	}
	if expiresAt.Before(time.Now()) {
		t.Fatalf("expected expiresAt in future")
	}

	claims, err := svc.ParseUserJWT(token)
	if err != nil {
		t.Fatalf("ParseUserJWT returned error: %v", err)
	}
	if claims.UserID != user.ID {
		t.Fatalf("claims user id want %d got %d", user.ID, claims.UserID)
	}

	var identity models.UserOAuthIdentity
	if err := db.Where("provider = ? AND provider_user_id = ?", constants.UserOAuthProviderTelegram, "987654").First(&identity).Error; err != nil {
		t.Fatalf("load identity failed: %v", err)
	}
	if identity.UserID != user.ID {
		t.Fatalf("identity user id want %d got %d", user.ID, identity.UserID)
	}
	if identity.Username != "mini_buyer" {
		t.Fatalf("identity username mismatch: %s", identity.Username)
	}
}
