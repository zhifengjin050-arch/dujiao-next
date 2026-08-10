package service

import (
	"context"
	"errors"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/dujiao-next/internal/config"
)

func TestVerifyMiniAppInitDataSuccess(t *testing.T) {
	svc := NewTelegramAuthService(config.TelegramAuthConfig{
		Enabled:            true,
		BotToken:           "test-bot-token",
		LoginExpireSeconds: 300,
		ReplayTTLSeconds:   300,
	})
	svc.replaySetNX = func(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
		return true, nil
	}

	initData := buildTestTelegramMiniAppInitData(t, "test-bot-token", time.Now().Unix(), `{"id":123456,"first_name":"Mini","last_name":"App","username":"mini_app","photo_url":"https://example.com/avatar.png"}`)
	verified, err := svc.VerifyMiniAppInitData(context.Background(), initData)
	if err != nil {
		t.Fatalf("VerifyMiniAppInitData returned error: %v", err)
	}
	if verified.ProviderUserID != "123456" {
		t.Fatalf("provider user id want 123456 got %s", verified.ProviderUserID)
	}
	if verified.FirstName != "Mini" || verified.LastName != "App" {
		t.Fatalf("unexpected name: %+v", verified)
	}
	if verified.Username != "mini_app" {
		t.Fatalf("username want mini_app got %s", verified.Username)
	}
	if verified.AvatarURL != "https://example.com/avatar.png" {
		t.Fatalf("avatar url mismatch: %s", verified.AvatarURL)
	}
}

func TestVerifyMiniAppInitDataExpired(t *testing.T) {
	svc := NewTelegramAuthService(config.TelegramAuthConfig{
		Enabled:            true,
		BotToken:           "test-bot-token",
		LoginExpireSeconds: 60,
		ReplayTTLSeconds:   60,
	})
	svc.replaySetNX = func(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
		return true, nil
	}

	initData := buildTestTelegramMiniAppInitData(t, "test-bot-token", time.Now().Add(-2*time.Minute).Unix(), `{"id":123456,"first_name":"Mini"}`)
	_, err := svc.VerifyMiniAppInitData(context.Background(), initData)
	if !errors.Is(err, ErrTelegramAuthExpired) {
		t.Fatalf("expected ErrTelegramAuthExpired, got %v", err)
	}
}

func TestVerifyMiniAppInitDataRejectsInvalidSignature(t *testing.T) {
	svc := NewTelegramAuthService(config.TelegramAuthConfig{
		Enabled:            true,
		BotToken:           "test-bot-token",
		LoginExpireSeconds: 300,
		ReplayTTLSeconds:   300,
	})
	svc.replaySetNX = func(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
		return true, nil
	}

	values := buildTestTelegramMiniAppValues(time.Now().Unix(), `{"id":123456,"first_name":"Mini"}`)
	values.Set("hash", "deadbeef")
	_, err := svc.VerifyMiniAppInitData(context.Background(), values.Encode())
	if !errors.Is(err, ErrTelegramAuthSignatureInvalid) {
		t.Fatalf("expected ErrTelegramAuthSignatureInvalid, got %v", err)
	}
}

func TestVerifyMiniAppInitDataAcceptsSignatureField(t *testing.T) {
	svc := NewTelegramAuthService(config.TelegramAuthConfig{
		Enabled:            true,
		BotToken:           "test-bot-token",
		LoginExpireSeconds: 300,
		ReplayTTLSeconds:   300,
	})
	svc.replaySetNX = func(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
		return true, nil
	}

	values := buildTestTelegramMiniAppValues(time.Now().Unix(), `{"id":123456,"first_name":"Mini","username":"mini_app"}`)
	values.Set("signature", "third-party-signature")
	values.Set("hash", buildTelegramMiniAppHash("test-bot-token", buildTelegramMiniAppDataCheckString(values)))

	verified, err := svc.VerifyMiniAppInitData(context.Background(), values.Encode())
	if err != nil {
		t.Fatalf("expected signature field initData to pass, got %v", err)
	}
	if verified.ProviderUserID != "123456" {
		t.Fatalf("provider user id want 123456 got %s", verified.ProviderUserID)
	}
}

func TestVerifyMiniAppInitDataRejectsReplay(t *testing.T) {
	svc := NewTelegramAuthService(config.TelegramAuthConfig{
		Enabled:            true,
		BotToken:           "test-bot-token",
		LoginExpireSeconds: 300,
		ReplayTTLSeconds:   300,
	})
	seen := map[string]bool{}
	svc.replaySetNX = func(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
		if seen[key] {
			return false, nil
		}
		seen[key] = true
		return true, nil
	}

	initData := buildTestTelegramMiniAppInitData(t, "test-bot-token", time.Now().Unix(), `{"id":123456,"first_name":"Mini"}`)
	if _, err := svc.VerifyMiniAppInitData(context.Background(), initData); err != nil {
		t.Fatalf("first VerifyMiniAppInitData returned error: %v", err)
	}
	_, err := svc.VerifyMiniAppInitData(context.Background(), initData)
	if !errors.Is(err, ErrTelegramAuthReplay) {
		t.Fatalf("expected ErrTelegramAuthReplay, got %v", err)
	}
}

func TestTelegramAuthServicePublicConfigIncludesMiniAppURL(t *testing.T) {
	svc := NewTelegramAuthService(config.TelegramAuthConfig{
		Enabled:     true,
		BotUsername: "demo_bot",
		MiniAppURL:  " https://example.com/mini-app ",
	})

	publicConfig := svc.PublicConfig()
	if publicConfig["enabled"] != true {
		t.Fatalf("expected enabled true, got %v", publicConfig["enabled"])
	}
	if publicConfig["bot_username"] != "demo_bot" {
		t.Fatalf("expected bot_username demo_bot, got %v", publicConfig["bot_username"])
	}
	if publicConfig["mini_app_url"] != "https://example.com/mini-app" {
		t.Fatalf("expected mini_app_url https://example.com/mini-app, got %v", publicConfig["mini_app_url"])
	}
}

func buildTestTelegramMiniAppInitData(t *testing.T, botToken string, authDate int64, userJSON string) string {
	t.Helper()
	values := buildTestTelegramMiniAppValues(authDate, userJSON)
	values.Set("hash", buildTelegramMiniAppHash(botToken, buildTelegramMiniAppDataCheckString(values)))
	return values.Encode()
}

func buildTestTelegramMiniAppValues(authDate int64, userJSON string) url.Values {
	values := url.Values{}
	values.Set("auth_date", strconv.FormatInt(authDate, 10))
	values.Set("query_id", "AAHdF6IQAAAAAN0XohDhrOrc")
	values.Set("user", userJSON)
	return values
}
