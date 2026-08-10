package service

import (
	"fmt"
	"strings"

	"github.com/dujiao-next/internal/config"
	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
)

// TelegramAuthSetting Telegram ??????
type TelegramAuthSetting struct {
	Enabled            bool   `json:"enabled"`
	BotUsername        string `json:"bot_username"`
	BotToken           string `json:"bot_token"`
	MiniAppURL         string `json:"mini_app_url"`
	LoginExpireSeconds int    `json:"login_expire_seconds"`
	ReplayTTLSeconds   int    `json:"replay_ttl_seconds"`
}

// TelegramAuthSettingPatch Telegram ??????
type TelegramAuthSettingPatch struct {
	Enabled            *bool   `json:"enabled"`
	BotUsername        *string `json:"bot_username"`
	BotToken           *string `json:"bot_token"`
	MiniAppURL         *string `json:"mini_app_url"`
	LoginExpireSeconds *int    `json:"login_expire_seconds"`
	ReplayTTLSeconds   *int    `json:"replay_ttl_seconds"`
}

// TelegramAuthDefaultSetting ?????????????
func TelegramAuthDefaultSetting(cfg config.TelegramAuthConfig) TelegramAuthSetting {
	return NormalizeTelegramAuthSetting(TelegramAuthSetting{
		Enabled:            cfg.Enabled,
		BotUsername:        strings.TrimSpace(cfg.BotUsername),
		BotToken:           strings.TrimSpace(cfg.BotToken),
		MiniAppURL:         strings.TrimSpace(cfg.MiniAppURL),
		LoginExpireSeconds: cfg.LoginExpireSeconds,
		ReplayTTLSeconds:   cfg.ReplayTTLSeconds,
	})
}

// NormalizeTelegramAuthSetting ??? Telegram ??
func NormalizeTelegramAuthSetting(setting TelegramAuthSetting) TelegramAuthSetting {
	setting.BotUsername = strings.TrimPrefix(strings.TrimSpace(setting.BotUsername), "@")
	setting.BotToken = strings.TrimSpace(setting.BotToken)
	setting.MiniAppURL = strings.TrimSpace(setting.MiniAppURL)

	if setting.LoginExpireSeconds <= 0 {
		setting.LoginExpireSeconds = 300
	}
	if setting.LoginExpireSeconds < 30 {
		setting.LoginExpireSeconds = 30
	}
	if setting.LoginExpireSeconds > 86400 {
		setting.LoginExpireSeconds = 86400
	}

	if setting.ReplayTTLSeconds <= 0 {
		setting.ReplayTTLSeconds = setting.LoginExpireSeconds
	}
	if setting.ReplayTTLSeconds < 60 {
		setting.ReplayTTLSeconds = 60
	}
	if setting.ReplayTTLSeconds > 86400 {
		setting.ReplayTTLSeconds = 86400
	}
	return setting
}

// ValidateTelegramAuthSetting ?? Telegram ?????
func ValidateTelegramAuthSetting(setting TelegramAuthSetting) error {
	normalized := NormalizeTelegramAuthSetting(setting)

	if normalized.LoginExpireSeconds < 30 || normalized.LoginExpireSeconds > 86400 {
		return fmt.Errorf("%w: ??????? 30-86400 ???", ErrTelegramAuthConfigInvalid)
	}
	if normalized.ReplayTTLSeconds < 60 || normalized.ReplayTTLSeconds > 86400 {
		return fmt.Errorf("%w: ???????? 60-86400 ???", ErrTelegramAuthConfigInvalid)
	}
	if !normalized.Enabled {
		return nil
	}
	if normalized.BotUsername == "" {
		return fmt.Errorf("%w: Bot ???????", ErrTelegramAuthConfigInvalid)
	}
	if strings.ContainsAny(normalized.BotUsername, " \t\r\n") {
		return fmt.Errorf("%w: Bot ???????", ErrTelegramAuthConfigInvalid)
	}
	if normalized.BotToken == "" {
		return fmt.Errorf("%w: Bot Token ????", ErrTelegramAuthConfigInvalid)
	}
	return nil
}

// TelegramAuthSettingToConfig ????????
func TelegramAuthSettingToConfig(setting TelegramAuthSetting) config.TelegramAuthConfig {
	normalized := NormalizeTelegramAuthSetting(setting)
	return config.TelegramAuthConfig{
		Enabled:            normalized.Enabled,
		BotUsername:        normalized.BotUsername,
		BotToken:           normalized.BotToken,
		MiniAppURL:         normalized.MiniAppURL,
		LoginExpireSeconds: normalized.LoginExpireSeconds,
		ReplayTTLSeconds:   normalized.ReplayTTLSeconds,
	}
}

// TelegramAuthSettingToMap ??? settings ????
func TelegramAuthSettingToMap(setting TelegramAuthSetting) map[string]interface{} {
	normalized := NormalizeTelegramAuthSetting(setting)
	return map[string]interface{}{
		"enabled":              normalized.Enabled,
		"bot_username":         normalized.BotUsername,
		"bot_token":            normalized.BotToken,
		"mini_app_url":         normalized.MiniAppURL,
		"login_expire_seconds": normalized.LoginExpireSeconds,
		"replay_ttl_seconds":   normalized.ReplayTTLSeconds,
	}
}

// MaskTelegramAuthSettingForAdmin ??????
func MaskTelegramAuthSettingForAdmin(setting TelegramAuthSetting) models.JSON {
	normalized := NormalizeTelegramAuthSetting(setting)
	return models.JSON{
		"enabled":              normalized.Enabled,
		"bot_username":         normalized.BotUsername,
		"bot_token":            "",
		"has_bot_token":        normalized.BotToken != "",
		"mini_app_url":         normalized.MiniAppURL,
		"login_expire_seconds": normalized.LoginExpireSeconds,
		"replay_ttl_seconds":   normalized.ReplayTTLSeconds,
	}
}

// GetTelegramAuthSetting ?? Telegram ????
func (s *SettingService) GetTelegramAuthSetting(defaultCfg config.TelegramAuthConfig) (TelegramAuthSetting, error) {
	fallback := TelegramAuthDefaultSetting(defaultCfg)
	value, err := s.GetByKey(constants.SettingKeyTelegramAuthConfig)
	if err != nil {
		return fallback, err
	}
	if value == nil {
		return fallback, nil
	}
	parsed := telegramAuthSettingFromJSON(value, fallback)
	return NormalizeTelegramAuthSetting(parsed), nil
}

// PatchTelegramAuthSetting ?????? Telegram ????
func (s *SettingService) PatchTelegramAuthSetting(defaultCfg config.TelegramAuthConfig, patch TelegramAuthSettingPatch) (TelegramAuthSetting, error) {
	current, err := s.GetTelegramAuthSetting(defaultCfg)
	if err != nil {
		return TelegramAuthSetting{}, err
	}

	next := current
	if patch.Enabled != nil {
		next.Enabled = *patch.Enabled
	}
	if patch.BotUsername != nil {
		next.BotUsername = strings.TrimSpace(*patch.BotUsername)
	}
	if patch.BotToken != nil {
		botToken := strings.TrimSpace(*patch.BotToken)
		if botToken != "" {
			next.BotToken = botToken
		}
	}
	if patch.MiniAppURL != nil {
		next.MiniAppURL = strings.TrimSpace(*patch.MiniAppURL)
	}
	if patch.LoginExpireSeconds != nil {
		next.LoginExpireSeconds = *patch.LoginExpireSeconds
	}
	if patch.ReplayTTLSeconds != nil {
		next.ReplayTTLSeconds = *patch.ReplayTTLSeconds
	}

	normalized := NormalizeTelegramAuthSetting(next)
	if err := ValidateTelegramAuthSetting(normalized); err != nil {
		return TelegramAuthSetting{}, err
	}
	if _, err := s.Update(constants.SettingKeyTelegramAuthConfig, TelegramAuthSettingToMap(normalized)); err != nil {
		return TelegramAuthSetting{}, err
	}
	return normalized, nil
}

func telegramAuthSettingFromJSON(raw models.JSON, fallback TelegramAuthSetting) TelegramAuthSetting {
	next := fallback
	if raw == nil {
		return next
	}
	next.Enabled = readBool(raw, "enabled", next.Enabled)
	next.BotUsername = readString(raw, "bot_username", next.BotUsername)
	next.BotToken = readString(raw, "bot_token", next.BotToken)
	next.MiniAppURL = readString(raw, "mini_app_url", next.MiniAppURL)
	next.LoginExpireSeconds = readInt(raw, "login_expire_seconds", next.LoginExpireSeconds)
	next.ReplayTTLSeconds = readInt(raw, "replay_ttl_seconds", next.ReplayTTLSeconds)
	return next
}
