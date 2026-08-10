package service

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/dujiao-next/internal/config"
	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/repository"
	"github.com/dujiao-next/internal/telegramidentity"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupTelegramOAuthTestService(t *testing.T) (*UserAuthService, *gorm.DB) {
	t.Helper()

	dsn := fmt.Sprintf("file:user_auth_service_oauth_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.UserOAuthIdentity{}, &models.EmailVerifyCode{}, &models.Setting{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	cfg := &config.Config{
		UserJWT: config.JWTConfig{
			SecretKey:   "user-jwt-test-secret",
			ExpireHours: 24,
		},
	}
	settingSvc := NewSettingService(repository.NewSettingRepository(db))
	svc := NewUserAuthService(
		cfg,
		repository.NewUserRepository(db),
		repository.NewUserOAuthIdentityRepository(db),
		repository.NewEmailVerifyCodeRepository(db),
		settingSvc,
		nil,
		nil,
	)
	return svc, db
}

func TestFindOrCreateTelegramUserRespectsRegistrationSetting(t *testing.T) {
	svc, db := setupTelegramOAuthTestService(t)

	if _, err := svc.settingService.Update(constants.SettingKeyRegistrationConfig, map[string]interface{}{
		constants.SettingFieldRegistrationEnabled: false,
	}); err != nil {
		t.Fatalf("disable registration failed: %v", err)
	}

	user, err := svc.findOrCreateTelegramUser(&TelegramIdentityVerified{
		Provider:       constants.UserOAuthProviderTelegram,
		ProviderUserID: "10001",
		Username:       "tg_new_user",
		AuthAt:         time.Now(),
	})
	if !errors.Is(err, ErrRegistrationDisabled) {
		t.Fatalf("expected ErrRegistrationDisabled, got user=%v err=%v", user, err)
	}

	var count int64
	if err := db.Model(&models.User{}).Count(&count).Error; err != nil {
		t.Fatalf("count users failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected no users created, got %d", count)
	}
}

func TestLoginWithTelegramAllowsExistingIdentityWhenRegistrationDisabled(t *testing.T) {
	svc, db := setupTelegramOAuthTestService(t)

	now := time.Now()
	user := &models.User{
		Email:       telegramidentity.BuildPlaceholderEmail("10002"),
		PasswordHash: "telegram-auto",
		DisplayName: "TG Existing",
		Status:      constants.UserStatusActive,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	identity := &models.UserOAuthIdentity{
		UserID:         user.ID,
		Provider:       constants.UserOAuthProviderTelegram,
		ProviderUserID: "10002",
		Username:       "tg_existing",
		AuthAt:         &now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := db.Create(identity).Error; err != nil {
		t.Fatalf("create identity failed: %v", err)
	}
	if _, err := svc.settingService.Update(constants.SettingKeyRegistrationConfig, map[string]interface{}{
		constants.SettingFieldRegistrationEnabled: false,
	}); err != nil {
		t.Fatalf("disable registration failed: %v", err)
	}

	gotUser, token, expiresAt, err := svc.loginWithVerifiedTelegram(&TelegramIdentityVerified{
		Provider:       constants.UserOAuthProviderTelegram,
		ProviderUserID: "10002",
		Username:       "tg_existing",
		AuthAt:         time.Now(),
	})
	if err != nil {
		t.Fatalf("loginWithVerifiedTelegram returned error: %v", err)
	}
	if gotUser == nil || gotUser.ID != user.ID {
		t.Fatalf("expected existing user %d, got %+v", user.ID, gotUser)
	}
	if token == "" {
		t.Fatalf("expected token")
	}
	if expiresAt.Before(time.Now()) {
		t.Fatalf("expected future expiresAt")
	}
}

func TestTelegramMiniAppLoginReturnsRegistrationDisabledWhenCreatingNewUser(t *testing.T) {
	svc, _ := setupTelegramOAuthTestService(t)
	telegramSvc := NewTelegramAuthService(config.TelegramAuthConfig{
		Enabled:            true,
		BotToken:           "test-bot-token",
		LoginExpireSeconds: 300,
		ReplayTTLSeconds:   300,
	})
	telegramSvc.replaySetNX = func(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
		return true, nil
	}
	svc.telegramAuthService = telegramSvc

	if _, err := svc.settingService.Update(constants.SettingKeyRegistrationConfig, map[string]interface{}{
		constants.SettingFieldRegistrationEnabled: false,
	}); err != nil {
		t.Fatalf("disable registration failed: %v", err)
	}

	initData := buildTestTelegramMiniAppInitData(t, "test-bot-token", time.Now().Unix(), `{"id":10003,"first_name":"Mini","last_name":"Blocked","username":"mini_blocked"}`)
	user, token, expiresAt, err := svc.LoginWithTelegramMiniApp(LoginWithTelegramMiniAppInput{
		InitData: initData,
		Context:  context.Background(),
	})
	if !errors.Is(err, ErrRegistrationDisabled) {
		t.Fatalf("expected ErrRegistrationDisabled, got user=%v token=%q expiresAt=%v err=%v", user, token, expiresAt, err)
	}
}
