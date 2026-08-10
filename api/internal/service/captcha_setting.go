package service

import (
	"fmt"
	"strings"

	"github.com/dujiao-next/internal/config"
	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
)

// CaptchaSceneSetting ???????
// ??:???????? 5 ???
// login ?????????????????????
// ????????????????????????????
// ????? settings ??????????
// ?????????????
//
//nolint:govet
type CaptchaSceneSetting struct {
	Login            bool `json:"login"`
	RegisterSendCode bool `json:"register_send_code"`
	ResetSendCode    bool `json:"reset_send_code"`
	GuestCreateOrder bool `json:"guest_create_order"`
	GiftCardRedeem   bool `json:"gift_card_redeem"`
}

// CaptchaImageSetting ???????
type CaptchaImageSetting struct {
	Length        int `json:"length"`
	Width         int `json:"width"`
	Height        int `json:"height"`
	NoiseCount    int `json:"noise_count"`
	ShowLine      int `json:"show_line"`
	ExpireSeconds int `json:"expire_seconds"`
	MaxStore      int `json:"max_store"`
}

// CaptchaTurnstileSetting Turnstile ??
type CaptchaTurnstileSetting struct {
	SiteKey   string `json:"site_key"`
	SecretKey string `json:"secret_key"`
	VerifyURL string `json:"verify_url"`
	TimeoutMS int    `json:"timeout_ms"`
}

// CaptchaSetting ???????
type CaptchaSetting struct {
	Provider  string                  `json:"provider"`
	Scenes    CaptchaSceneSetting     `json:"scenes"`
	Image     CaptchaImageSetting     `json:"image"`
	Turnstile CaptchaTurnstileSetting `json:"turnstile"`
}

// CaptchaScenePatch ??????
type CaptchaScenePatch struct {
	Login            *bool `json:"login"`
	RegisterSendCode *bool `json:"register_send_code"`
	ResetSendCode    *bool `json:"reset_send_code"`
	GuestCreateOrder *bool `json:"guest_create_order"`
	GiftCardRedeem   *bool `json:"gift_card_redeem"`
}

// CaptchaImagePatch ??????
type CaptchaImagePatch struct {
	Length        *int `json:"length"`
	Width         *int `json:"width"`
	Height        *int `json:"height"`
	NoiseCount    *int `json:"noise_count"`
	ShowLine      *int `json:"show_line"`
	ExpireSeconds *int `json:"expire_seconds"`
	MaxStore      *int `json:"max_store"`
}

// CaptchaTurnstilePatch Turnstile ????
type CaptchaTurnstilePatch struct {
	SiteKey   *string `json:"site_key"`
	SecretKey *string `json:"secret_key"`
	VerifyURL *string `json:"verify_url"`
	TimeoutMS *int    `json:"timeout_ms"`
}

// CaptchaSettingPatch ???????
type CaptchaSettingPatch struct {
	Provider  *string                `json:"provider"`
	Scenes    *CaptchaScenePatch     `json:"scenes"`
	Image     *CaptchaImagePatch     `json:"image"`
	Turnstile *CaptchaTurnstilePatch `json:"turnstile"`
}

// CaptchaDefaultSetting ???????????????
func CaptchaDefaultSetting(cfg config.CaptchaConfig) CaptchaSetting {
	setting := CaptchaSetting{
		Provider: strings.ToLower(strings.TrimSpace(cfg.Provider)),
		Scenes: CaptchaSceneSetting{
			Login:            cfg.Scenes.Login,
			RegisterSendCode: cfg.Scenes.RegisterSendCode,
			ResetSendCode:    cfg.Scenes.ResetSendCode,
			GuestCreateOrder: cfg.Scenes.GuestCreateOrder,
			GiftCardRedeem:   cfg.Scenes.GiftCardRedeem,
		},
		Image: CaptchaImageSetting{
			Length:        cfg.Image.Length,
			Width:         cfg.Image.Width,
			Height:        cfg.Image.Height,
			NoiseCount:    cfg.Image.NoiseCount,
			ShowLine:      cfg.Image.ShowLine,
			ExpireSeconds: cfg.Image.ExpireSeconds,
			MaxStore:      cfg.Image.MaxStore,
		},
		Turnstile: CaptchaTurnstileSetting{
			SiteKey:   strings.TrimSpace(cfg.Turnstile.SiteKey),
			SecretKey: strings.TrimSpace(cfg.Turnstile.SecretKey),
			VerifyURL: strings.TrimSpace(cfg.Turnstile.VerifyURL),
			TimeoutMS: cfg.Turnstile.TimeoutMS,
		},
	}
	return NormalizeCaptchaSetting(setting)
}

// NormalizeCaptchaSetting ????????
func NormalizeCaptchaSetting(setting CaptchaSetting) CaptchaSetting {
	provider := strings.ToLower(strings.TrimSpace(setting.Provider))
	switch provider {
	case constants.CaptchaProviderImage, constants.CaptchaProviderTurnstile, constants.CaptchaProviderNone:
		setting.Provider = provider
	default:
		setting.Provider = constants.CaptchaProviderNone
	}

	if setting.Image.Length < 4 || setting.Image.Length > 8 {
		setting.Image.Length = 5
	}
	if setting.Image.Width < 100 {
		setting.Image.Width = 240
	}
	if setting.Image.Height < 40 {
		setting.Image.Height = 80
	}
	if setting.Image.NoiseCount < 0 {
		setting.Image.NoiseCount = 2
	}
	if setting.Image.ShowLine < 0 {
		setting.Image.ShowLine = 2
	}
	if setting.Image.ExpireSeconds < 30 || setting.Image.ExpireSeconds > 3600 {
		setting.Image.ExpireSeconds = 300
	}
	if setting.Image.MaxStore < 100 {
		setting.Image.MaxStore = 10240
	}

	setting.Turnstile.SiteKey = strings.TrimSpace(setting.Turnstile.SiteKey)
	setting.Turnstile.SecretKey = strings.TrimSpace(setting.Turnstile.SecretKey)
	setting.Turnstile.VerifyURL = strings.TrimSpace(setting.Turnstile.VerifyURL)
	if setting.Turnstile.VerifyURL == "" {
		setting.Turnstile.VerifyURL = "https://challenges.cloudflare.com/turnstile/v0/siteverify"
	}
	if setting.Turnstile.TimeoutMS <= 0 {
		setting.Turnstile.TimeoutMS = 2000
	}

	return setting
}

// ValidateCaptchaSetting ???????
func ValidateCaptchaSetting(setting CaptchaSetting) error {
	normalized := NormalizeCaptchaSetting(setting)

	switch normalized.Provider {
	case constants.CaptchaProviderNone, constants.CaptchaProviderImage, constants.CaptchaProviderTurnstile:
	default:
		return fmt.Errorf("%w: ????????", ErrCaptchaConfigInvalid)
	}

	if normalized.Provider == constants.CaptchaProviderNone && normalized.Scenes.anyEnabled() {
		return fmt.Errorf("%w: ???????????????????", ErrCaptchaConfigInvalid)
	}

	if normalized.Provider == constants.CaptchaProviderTurnstile {
		if strings.TrimSpace(normalized.Turnstile.SiteKey) == "" {
			return fmt.Errorf("%w: Turnstile Site Key ????", ErrCaptchaConfigInvalid)
		}
		if strings.TrimSpace(normalized.Turnstile.SecretKey) == "" {
			return fmt.Errorf("%w: Turnstile Secret Key ????", ErrCaptchaConfigInvalid)
		}
	}

	if normalized.Image.Length < 4 || normalized.Image.Length > 8 {
		return fmt.Errorf("%w: ????????? 4-8 ??", ErrCaptchaConfigInvalid)
	}
	if normalized.Image.Width < 100 || normalized.Image.Height < 40 {
		return fmt.Errorf("%w: ??????????", ErrCaptchaConfigInvalid)
	}
	if normalized.Image.ExpireSeconds < 30 || normalized.Image.ExpireSeconds > 3600 {
		return fmt.Errorf("%w: ??????????? 30-3600 ?", ErrCaptchaConfigInvalid)
	}
	if normalized.Turnstile.TimeoutMS < 500 || normalized.Turnstile.TimeoutMS > 10000 {
		return fmt.Errorf("%w: Turnstile ?????? 500-10000ms", ErrCaptchaConfigInvalid)
	}

	return nil
}

// CaptchaSettingToConfig ? settings ??????????
func CaptchaSettingToConfig(setting CaptchaSetting) config.CaptchaConfig {
	normalized := NormalizeCaptchaSetting(setting)
	return config.CaptchaConfig{
		Provider: normalized.Provider,
		Scenes: config.CaptchaSceneConfig{
			Login:            normalized.Scenes.Login,
			RegisterSendCode: normalized.Scenes.RegisterSendCode,
			ResetSendCode:    normalized.Scenes.ResetSendCode,
			GuestCreateOrder: normalized.Scenes.GuestCreateOrder,
			GiftCardRedeem:   normalized.Scenes.GiftCardRedeem,
		},
		Image: config.CaptchaImageConfig{
			Length:        normalized.Image.Length,
			Width:         normalized.Image.Width,
			Height:        normalized.Image.Height,
			NoiseCount:    normalized.Image.NoiseCount,
			ShowLine:      normalized.Image.ShowLine,
			ExpireSeconds: normalized.Image.ExpireSeconds,
			MaxStore:      normalized.Image.MaxStore,
		},
		Turnstile: config.CaptchaTurnstileConfig{
			SiteKey:   normalized.Turnstile.SiteKey,
			SecretKey: normalized.Turnstile.SecretKey,
			VerifyURL: normalized.Turnstile.VerifyURL,
			TimeoutMS: normalized.Turnstile.TimeoutMS,
		},
	}
}

// CaptchaSettingToMap ????????? settings ???
func CaptchaSettingToMap(setting CaptchaSetting) map[string]interface{} {
	normalized := NormalizeCaptchaSetting(setting)
	return map[string]interface{}{
		"provider": normalized.Provider,
		"scenes": map[string]interface{}{
			"login":              normalized.Scenes.Login,
			"register_send_code": normalized.Scenes.RegisterSendCode,
			"reset_send_code":    normalized.Scenes.ResetSendCode,
			"guest_create_order": normalized.Scenes.GuestCreateOrder,
			"gift_card_redeem":   normalized.Scenes.GiftCardRedeem,
		},
		"image": map[string]interface{}{
			"length":         normalized.Image.Length,
			"width":          normalized.Image.Width,
			"height":         normalized.Image.Height,
			"noise_count":    normalized.Image.NoiseCount,
			"show_line":      normalized.Image.ShowLine,
			"expire_seconds": normalized.Image.ExpireSeconds,
			"max_store":      normalized.Image.MaxStore,
		},
		"turnstile": map[string]interface{}{
			"site_key":   normalized.Turnstile.SiteKey,
			"secret_key": normalized.Turnstile.SecretKey,
			"verify_url": normalized.Turnstile.VerifyURL,
			"timeout_ms": normalized.Turnstile.TimeoutMS,
		},
	}
}

// MaskCaptchaSettingForAdmin ???????????
func MaskCaptchaSettingForAdmin(setting CaptchaSetting) models.JSON {
	normalized := NormalizeCaptchaSetting(setting)
	return models.JSON{
		"provider": normalized.Provider,
		"scenes": map[string]interface{}{
			"login":              normalized.Scenes.Login,
			"register_send_code": normalized.Scenes.RegisterSendCode,
			"reset_send_code":    normalized.Scenes.ResetSendCode,
			"guest_create_order": normalized.Scenes.GuestCreateOrder,
			"gift_card_redeem":   normalized.Scenes.GiftCardRedeem,
		},
		"image": map[string]interface{}{
			"length":         normalized.Image.Length,
			"width":          normalized.Image.Width,
			"height":         normalized.Image.Height,
			"noise_count":    normalized.Image.NoiseCount,
			"show_line":      normalized.Image.ShowLine,
			"expire_seconds": normalized.Image.ExpireSeconds,
			"max_store":      normalized.Image.MaxStore,
		},
		"turnstile": map[string]interface{}{
			"site_key":   normalized.Turnstile.SiteKey,
			"secret_key": "",
			"has_secret": normalized.Turnstile.SecretKey != "",
			"verify_url": normalized.Turnstile.VerifyURL,
			"timeout_ms": normalized.Turnstile.TimeoutMS,
		},
	}
}

// PublicCaptchaSetting ???????????????
func PublicCaptchaSetting(setting CaptchaSetting) models.JSON {
	normalized := NormalizeCaptchaSetting(setting)
	public := models.JSON{
		"provider": normalized.Provider,
		"scenes": map[string]interface{}{
			"login":              normalized.Scenes.Login,
			"register_send_code": normalized.Scenes.RegisterSendCode,
			"reset_send_code":    normalized.Scenes.ResetSendCode,
			"guest_create_order": normalized.Scenes.GuestCreateOrder,
			"gift_card_redeem":   normalized.Scenes.GiftCardRedeem,
		},
	}
	if normalized.Provider == constants.CaptchaProviderTurnstile {
		public["turnstile"] = map[string]interface{}{
			"site_key": normalized.Turnstile.SiteKey,
		}
	}
	return public
}

func (s CaptchaSceneSetting) anyEnabled() bool {
	return s.Login || s.RegisterSendCode || s.ResetSendCode || s.GuestCreateOrder || s.GiftCardRedeem
}

// IsSceneEnabled ??????????
func (s CaptchaSetting) IsSceneEnabled(scene string) bool {
	switch strings.ToLower(strings.TrimSpace(scene)) {
	case constants.CaptchaSceneLogin:
		return s.Scenes.Login
	case constants.CaptchaSceneRegisterSendCode:
		return s.Scenes.RegisterSendCode
	case constants.CaptchaSceneResetSendCode:
		return s.Scenes.ResetSendCode
	case constants.CaptchaSceneGuestCreateOrder:
		return s.Scenes.GuestCreateOrder
	case constants.CaptchaSceneGiftCardRedeem:
		return s.Scenes.GiftCardRedeem
	default:
		return false
	}
}

// GetCaptchaSetting ???????(?? settings,???? config.yml)
func (s *SettingService) GetCaptchaSetting(defaultCfg config.CaptchaConfig) (CaptchaSetting, error) {
	fallback := CaptchaDefaultSetting(defaultCfg)
	value, err := s.GetByKey(constants.SettingKeyCaptchaConfig)
	if err != nil {
		return fallback, err
	}
	if value == nil {
		return fallback, nil
	}
	parsed := captchaSettingFromJSON(value, fallback)
	return NormalizeCaptchaSetting(parsed), nil
}

// PatchCaptchaSetting ???????????
func (s *SettingService) PatchCaptchaSetting(defaultCfg config.CaptchaConfig, patch CaptchaSettingPatch) (CaptchaSetting, error) {
	current, err := s.GetCaptchaSetting(defaultCfg)
	if err != nil {
		return CaptchaSetting{}, err
	}

	next := current
	if patch.Provider != nil {
		next.Provider = strings.ToLower(strings.TrimSpace(*patch.Provider))
	}
	if patch.Scenes != nil {
		if patch.Scenes.Login != nil {
			next.Scenes.Login = *patch.Scenes.Login
		}
		if patch.Scenes.RegisterSendCode != nil {
			next.Scenes.RegisterSendCode = *patch.Scenes.RegisterSendCode
		}
		if patch.Scenes.ResetSendCode != nil {
			next.Scenes.ResetSendCode = *patch.Scenes.ResetSendCode
		}
		if patch.Scenes.GuestCreateOrder != nil {
			next.Scenes.GuestCreateOrder = *patch.Scenes.GuestCreateOrder
		}
		if patch.Scenes.GiftCardRedeem != nil {
			next.Scenes.GiftCardRedeem = *patch.Scenes.GiftCardRedeem
		}
	}
	if patch.Image != nil {
		if patch.Image.Length != nil {
			next.Image.Length = *patch.Image.Length
		}
		if patch.Image.Width != nil {
			next.Image.Width = *patch.Image.Width
		}
		if patch.Image.Height != nil {
			next.Image.Height = *patch.Image.Height
		}
		if patch.Image.NoiseCount != nil {
			next.Image.NoiseCount = *patch.Image.NoiseCount
		}
		if patch.Image.ShowLine != nil {
			next.Image.ShowLine = *patch.Image.ShowLine
		}
		if patch.Image.ExpireSeconds != nil {
			next.Image.ExpireSeconds = *patch.Image.ExpireSeconds
		}
		if patch.Image.MaxStore != nil {
			next.Image.MaxStore = *patch.Image.MaxStore
		}
	}
	if patch.Turnstile != nil {
		if patch.Turnstile.SiteKey != nil {
			next.Turnstile.SiteKey = strings.TrimSpace(*patch.Turnstile.SiteKey)
		}
		if patch.Turnstile.SecretKey != nil {
			secret := strings.TrimSpace(*patch.Turnstile.SecretKey)
			if secret != "" {
				next.Turnstile.SecretKey = secret
			}
		}
		if patch.Turnstile.VerifyURL != nil {
			next.Turnstile.VerifyURL = strings.TrimSpace(*patch.Turnstile.VerifyURL)
		}
		if patch.Turnstile.TimeoutMS != nil {
			next.Turnstile.TimeoutMS = *patch.Turnstile.TimeoutMS
		}
	}

	normalized := NormalizeCaptchaSetting(next)
	if err := ValidateCaptchaSetting(normalized); err != nil {
		return CaptchaSetting{}, err
	}

	if _, err := s.Update(constants.SettingKeyCaptchaConfig, CaptchaSettingToMap(normalized)); err != nil {
		return CaptchaSetting{}, err
	}
	return normalized, nil
}

func captchaSettingFromJSON(raw models.JSON, fallback CaptchaSetting) CaptchaSetting {
	next := fallback
	if raw == nil {
		return next
	}

	next.Provider = readString(raw, "provider", next.Provider)

	scenesRaw, ok := raw["scenes"]
	if ok {
		if scenesMap := toStringAnyMap(scenesRaw); scenesMap != nil {
			next.Scenes.Login = readBool(scenesMap, "login", next.Scenes.Login)
			next.Scenes.RegisterSendCode = readBool(scenesMap, "register_send_code", next.Scenes.RegisterSendCode)
			next.Scenes.ResetSendCode = readBool(scenesMap, "reset_send_code", next.Scenes.ResetSendCode)
			next.Scenes.GuestCreateOrder = readBool(scenesMap, "guest_create_order", next.Scenes.GuestCreateOrder)
			next.Scenes.GiftCardRedeem = readBool(scenesMap, "gift_card_redeem", next.Scenes.GiftCardRedeem)
		}
	}

	imageRaw, ok := raw["image"]
	if ok {
		if imageMap := toStringAnyMap(imageRaw); imageMap != nil {
			next.Image.Length = readInt(imageMap, "length", next.Image.Length)
			next.Image.Width = readInt(imageMap, "width", next.Image.Width)
			next.Image.Height = readInt(imageMap, "height", next.Image.Height)
			next.Image.NoiseCount = readInt(imageMap, "noise_count", next.Image.NoiseCount)
			next.Image.ShowLine = readInt(imageMap, "show_line", next.Image.ShowLine)
			next.Image.ExpireSeconds = readInt(imageMap, "expire_seconds", next.Image.ExpireSeconds)
			next.Image.MaxStore = readInt(imageMap, "max_store", next.Image.MaxStore)
		}
	}

	turnstileRaw, ok := raw["turnstile"]
	if ok {
		if turnstileMap := toStringAnyMap(turnstileRaw); turnstileMap != nil {
			next.Turnstile.SiteKey = readString(turnstileMap, "site_key", next.Turnstile.SiteKey)
			next.Turnstile.SecretKey = readString(turnstileMap, "secret_key", next.Turnstile.SecretKey)
			next.Turnstile.VerifyURL = readString(turnstileMap, "verify_url", next.Turnstile.VerifyURL)
			next.Turnstile.TimeoutMS = readInt(turnstileMap, "timeout_ms", next.Turnstile.TimeoutMS)
		}
	}

	return next
}
