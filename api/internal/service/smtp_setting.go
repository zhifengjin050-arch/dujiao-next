package service

import (
	"fmt"
	"net/mail"
	"strings"

	"github.com/dujiao-next/internal/config"
	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
)

// SMTPVerifyCodeSetting SMTP ???????
// ??:??????? SMTP ??? verify_code ???
// ????? NormalizeSMTPSetting ? ValidateSMTPSetting ?????
// ????????????????
// ??:json tag ?????????
// ??????,?? nil ????
// ???? SMTPDefaultSetting ??
// ? config.VerifyCodeConfig ????
// ??? settings ???????
// ???????????
// ??????
// ????????????
// ????????????????
// ???????
// ?????
// ????????? config.EmailConfig
// ??? handler ?????
// ??????
// ?????
// ?????????????
// ??
//
// (??????????)
//
//nolint:govet
type SMTPVerifyCodeSetting struct {
	ExpireMinutes       int `json:"expire_minutes"`
	SendIntervalSeconds int `json:"send_interval_seconds"`
	MaxAttempts         int `json:"max_attempts"`
	Length              int `json:"length"`
}

// SMTPSetting SMTP ????
type SMTPSetting struct {
	Enabled                  bool                  `json:"enabled"`
	Host                     string                `json:"host"`
	Port                     int                   `json:"port"`
	Username                 string                `json:"username"`
	Password                 string                `json:"password"`
	From                     string                `json:"from"`
	FromName                 string                `json:"from_name"`
	UseTLS                   bool                  `json:"use_tls"`
	UseSSL                   bool                  `json:"use_ssl"`
	OrderNotificationEnabled bool                  `json:"order_notification_enabled"`
	VerifyCode               SMTPVerifyCodeSetting `json:"verify_code"`
}

// SMTPVerifyCodePatch SMTP ???????
type SMTPVerifyCodePatch struct {
	ExpireMinutes       *int `json:"expire_minutes"`
	SendIntervalSeconds *int `json:"send_interval_seconds"`
	MaxAttempts         *int `json:"max_attempts"`
	Length              *int `json:"length"`
}

// SMTPSettingPatch SMTP ????(??????)
type SMTPSettingPatch struct {
	Enabled                  *bool                `json:"enabled"`
	Host                     *string              `json:"host"`
	Port                     *int                 `json:"port"`
	Username                 *string              `json:"username"`
	Password                 *string              `json:"password"`
	From                     *string              `json:"from"`
	FromName                 *string              `json:"from_name"`
	UseTLS                   *bool                `json:"use_tls"`
	UseSSL                   *bool                `json:"use_ssl"`
	OrderNotificationEnabled *bool                `json:"order_notification_enabled"`
	VerifyCode               *SMTPVerifyCodePatch `json:"verify_code"`
}

// SMTPDefaultSetting ?????????? SMTP ??
func SMTPDefaultSetting(cfg config.EmailConfig) SMTPSetting {
	setting := SMTPSetting{
		Enabled:                  cfg.Enabled,
		Host:                     strings.TrimSpace(cfg.Host),
		Port:                     cfg.Port,
		Username:                 strings.TrimSpace(cfg.Username),
		Password:                 strings.TrimSpace(cfg.Password),
		From:                     strings.TrimSpace(cfg.From),
		FromName:                 strings.TrimSpace(cfg.FromName),
		UseTLS:                   cfg.UseTLS,
		UseSSL:                   cfg.UseSSL,
		OrderNotificationEnabled: true,
		VerifyCode: SMTPVerifyCodeSetting{
			ExpireMinutes:       cfg.VerifyCode.ExpireMinutes,
			SendIntervalSeconds: cfg.VerifyCode.SendIntervalSeconds,
			MaxAttempts:         cfg.VerifyCode.MaxAttempts,
			Length:              cfg.VerifyCode.Length,
		},
	}
	return NormalizeSMTPSetting(setting)
}

// NormalizeSMTPSetting ??? SMTP ????????
func NormalizeSMTPSetting(setting SMTPSetting) SMTPSetting {
	setting.Host = strings.TrimSpace(setting.Host)
	setting.Username = strings.TrimSpace(setting.Username)
	setting.Password = strings.TrimSpace(setting.Password)
	setting.From = strings.TrimSpace(setting.From)
	setting.FromName = strings.TrimSpace(setting.FromName)

	if setting.Port <= 0 || setting.Port > 65535 {
		setting.Port = 587
	}

	if setting.VerifyCode.ExpireMinutes <= 0 {
		setting.VerifyCode.ExpireMinutes = 10
	}
	if setting.VerifyCode.SendIntervalSeconds <= 0 {
		setting.VerifyCode.SendIntervalSeconds = 60
	}
	if setting.VerifyCode.MaxAttempts <= 0 {
		setting.VerifyCode.MaxAttempts = 5
	}
	if setting.VerifyCode.Length < 4 || setting.VerifyCode.Length > 10 {
		setting.VerifyCode.Length = 6
	}

	return setting
}

// ValidateSMTPSetting ?? SMTP ?????
func ValidateSMTPSetting(setting SMTPSetting) error {
	if setting.Port <= 0 || setting.Port > 65535 {
		return fmt.Errorf("%w: SMTP ????? 1-65535", ErrSMTPConfigInvalid)
	}
	if setting.UseTLS && setting.UseSSL {
		return fmt.Errorf("%w: TLS ? SSL ??????", ErrSMTPConfigInvalid)
	}
	if setting.VerifyCode.Length < 4 || setting.VerifyCode.Length > 10 {
		return fmt.Errorf("%w: ??????? 4-10 ??", ErrSMTPConfigInvalid)
	}
	if setting.VerifyCode.ExpireMinutes <= 0 {
		return fmt.Errorf("%w: ??????????? 0", ErrSMTPConfigInvalid)
	}
	if setting.VerifyCode.SendIntervalSeconds <= 0 {
		return fmt.Errorf("%w: ??????????? 0", ErrSMTPConfigInvalid)
	}
	if setting.VerifyCode.MaxAttempts <= 0 {
		return fmt.Errorf("%w: ??????????? 0", ErrSMTPConfigInvalid)
	}
	if !setting.Enabled {
		return nil
	}
	if strings.TrimSpace(setting.Host) == "" {
		return fmt.Errorf("%w: SMTP ??????", ErrSMTPConfigInvalid)
	}
	if strings.TrimSpace(setting.From) == "" {
		return fmt.Errorf("%w: ?????????", ErrSMTPConfigInvalid)
	}
	if _, err := mail.ParseAddress(setting.From); err != nil {
		return fmt.Errorf("%w: ?????????", ErrSMTPConfigInvalid)
	}
	return nil
}

// SMTPSettingToConfig ? SMTP ??????????
func SMTPSettingToConfig(setting SMTPSetting) config.EmailConfig {
	normalized := NormalizeSMTPSetting(setting)
	return config.EmailConfig{
		Enabled:  normalized.Enabled,
		Host:     normalized.Host,
		Port:     normalized.Port,
		Username: normalized.Username,
		Password: normalized.Password,
		From:     normalized.From,
		FromName: normalized.FromName,
		UseTLS:   normalized.UseTLS,
		UseSSL:   normalized.UseSSL,
		VerifyCode: config.VerifyCodeConfig{
			ExpireMinutes:       normalized.VerifyCode.ExpireMinutes,
			SendIntervalSeconds: normalized.VerifyCode.SendIntervalSeconds,
			MaxAttempts:         normalized.VerifyCode.MaxAttempts,
			Length:              normalized.VerifyCode.Length,
		},
	}
}

// SMTPSettingToMap ? SMTP ????? settings ???
func SMTPSettingToMap(setting SMTPSetting) map[string]interface{} {
	normalized := NormalizeSMTPSetting(setting)
	return map[string]interface{}{
		"enabled":                    normalized.Enabled,
		"host":                       normalized.Host,
		"port":                       normalized.Port,
		"username":                   normalized.Username,
		"password":                   normalized.Password,
		"from":                       normalized.From,
		"from_name":                  normalized.FromName,
		"use_tls":                    normalized.UseTLS,
		"use_ssl":                    normalized.UseSSL,
		"order_notification_enabled": normalized.OrderNotificationEnabled,
		"verify_code": map[string]interface{}{
			"expire_minutes":        normalized.VerifyCode.ExpireMinutes,
			"send_interval_seconds": normalized.VerifyCode.SendIntervalSeconds,
			"max_attempts":          normalized.VerifyCode.MaxAttempts,
			"length":                normalized.VerifyCode.Length,
		},
	}
}

// MaskSMTPSettingForAdmin ?????? SMTP ??
func MaskSMTPSettingForAdmin(setting SMTPSetting) models.JSON {
	normalized := NormalizeSMTPSetting(setting)
	return models.JSON{
		"enabled":                    normalized.Enabled,
		"host":                       normalized.Host,
		"port":                       normalized.Port,
		"username":                   normalized.Username,
		"password":                   "",
		"has_password":               normalized.Password != "",
		"from":                       normalized.From,
		"from_name":                  normalized.FromName,
		"use_tls":                    normalized.UseTLS,
		"use_ssl":                    normalized.UseSSL,
		"order_notification_enabled": normalized.OrderNotificationEnabled,
		"verify_code": map[string]interface{}{
			"expire_minutes":        normalized.VerifyCode.ExpireMinutes,
			"send_interval_seconds": normalized.VerifyCode.SendIntervalSeconds,
			"max_attempts":          normalized.VerifyCode.MaxAttempts,
			"length":                normalized.VerifyCode.Length,
		},
	}
}

// GetSMTPSetting ?? SMTP ??(?? settings,????????)
func (s *SettingService) GetSMTPSetting(defaultCfg config.EmailConfig) (SMTPSetting, error) {
	fallback := SMTPDefaultSetting(defaultCfg)
	value, err := s.GetByKey(constants.SettingKeySMTPConfig)
	if err != nil {
		return fallback, err
	}
	if value == nil {
		return fallback, nil
	}
	parsed := smtpSettingFromJSON(value, fallback)
	return NormalizeSMTPSetting(parsed), nil
}

// PatchSMTPSetting ?????? SMTP ??
func (s *SettingService) PatchSMTPSetting(defaultCfg config.EmailConfig, patch SMTPSettingPatch) (SMTPSetting, error) {
	current, err := s.GetSMTPSetting(defaultCfg)
	if err != nil {
		return SMTPSetting{}, err
	}

	next := current
	if patch.Enabled != nil {
		next.Enabled = *patch.Enabled
	}
	if patch.Host != nil {
		next.Host = strings.TrimSpace(*patch.Host)
	}
	if patch.Port != nil {
		next.Port = *patch.Port
	}
	if patch.Username != nil {
		next.Username = strings.TrimSpace(*patch.Username)
	}
	if patch.Password != nil {
		password := strings.TrimSpace(*patch.Password)
		if password != "" {
			next.Password = password
		}
	}
	if patch.From != nil {
		next.From = strings.TrimSpace(*patch.From)
	}
	if patch.FromName != nil {
		next.FromName = strings.TrimSpace(*patch.FromName)
	}
	if patch.UseTLS != nil {
		next.UseTLS = *patch.UseTLS
	}
	if patch.UseSSL != nil {
		next.UseSSL = *patch.UseSSL
	}
	if patch.OrderNotificationEnabled != nil {
		next.OrderNotificationEnabled = *patch.OrderNotificationEnabled
	}
	if patch.VerifyCode != nil {
		if patch.VerifyCode.ExpireMinutes != nil {
			next.VerifyCode.ExpireMinutes = *patch.VerifyCode.ExpireMinutes
		}
		if patch.VerifyCode.SendIntervalSeconds != nil {
			next.VerifyCode.SendIntervalSeconds = *patch.VerifyCode.SendIntervalSeconds
		}
		if patch.VerifyCode.MaxAttempts != nil {
			next.VerifyCode.MaxAttempts = *patch.VerifyCode.MaxAttempts
		}
		if patch.VerifyCode.Length != nil {
			next.VerifyCode.Length = *patch.VerifyCode.Length
		}
	}

	normalized := NormalizeSMTPSetting(next)
	if err := ValidateSMTPSetting(normalized); err != nil {
		return SMTPSetting{}, err
	}

	if _, err := s.Update(constants.SettingKeySMTPConfig, SMTPSettingToMap(normalized)); err != nil {
		return SMTPSetting{}, err
	}
	return normalized, nil
}

func smtpSettingFromJSON(raw models.JSON, fallback SMTPSetting) SMTPSetting {
	next := fallback
	if raw == nil {
		return next
	}

	next.Enabled = readBool(raw, "enabled", next.Enabled)
	next.Host = readString(raw, "host", next.Host)
	next.Port = readInt(raw, "port", next.Port)
	next.Username = readString(raw, "username", next.Username)
	next.Password = readString(raw, "password", next.Password)
	next.From = readString(raw, "from", next.From)
	next.FromName = readString(raw, "from_name", next.FromName)
	next.UseTLS = readBool(raw, "use_tls", next.UseTLS)
	next.UseSSL = readBool(raw, "use_ssl", next.UseSSL)
	next.OrderNotificationEnabled = readBool(raw, "order_notification_enabled", next.OrderNotificationEnabled)

	verifyRaw, ok := raw["verify_code"]
	if ok {
		if verifyMap := toStringAnyMap(verifyRaw); verifyMap != nil {
			next.VerifyCode.ExpireMinutes = readInt(verifyMap, "expire_minutes", next.VerifyCode.ExpireMinutes)
			next.VerifyCode.SendIntervalSeconds = readInt(verifyMap, "send_interval_seconds", next.VerifyCode.SendIntervalSeconds)
			next.VerifyCode.MaxAttempts = readInt(verifyMap, "max_attempts", next.VerifyCode.MaxAttempts)
			next.VerifyCode.Length = readInt(verifyMap, "length", next.VerifyCode.Length)
		}
	}

	return next
}
