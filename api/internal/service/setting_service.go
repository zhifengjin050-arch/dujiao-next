package service

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/dujiao-next/internal/config"
	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/repository"
)

const (
	orderConfigFieldPaymentExpireMinutes = "payment_expire_minutes"
	orderConfigFieldMaxRefundDays        = "max_refund_days"

	orderPaymentExpireMinutesDefault = 15
	orderPaymentExpireMinutesMin     = 1
	orderPaymentExpireMinutesMax     = 10080

	orderRefundMaxDaysDefault = 30
	orderRefundMaxDaysMin     = 0
	orderRefundMaxDaysMax     = 3650
)

// OrderConfig ?????
type OrderConfig struct {
	PaymentExpireMinutes int `json:"payment_expire_minutes"`
	MaxRefundDays        int `json:"max_refund_days"`
}

// OrderRefundConfig ???????
type OrderRefundConfig struct {
	MaxRefundDays int `json:"max_refund_days"`
}

// SettingService ??????
type SettingService struct {
	repo                  repository.SettingRepository
	defaultOrderConfig    config.OrderConfig
	hasDefaultOrderConfig bool
}

// SiteBrand ??????
type SiteBrand struct {
	SiteName string
	SiteURL  string
}

// NewSettingService ???????
// ???? order ????,??? settings ??????? config ? order ???
func NewSettingService(repo repository.SettingRepository, defaultOrderCfg ...config.OrderConfig) *SettingService {
	svc := &SettingService{repo: repo}
	if len(defaultOrderCfg) > 0 {
		svc.defaultOrderConfig = defaultOrderCfg[0]
		svc.hasDefaultOrderConfig = true
	}
	return svc
}

// GetConfig ??????(?????)
func (s *SettingService) GetConfig(defaults map[string]interface{}) (map[string]interface{}, error) {
	data := make(map[string]interface{})
	for k, v := range defaults {
		data[k] = v
	}

	setting, err := s.repo.GetByKey(constants.SettingKeySiteConfig)
	if err != nil {
		return nil, err
	}
	if setting == nil {
		return data, nil
	}

	for k, v := range setting.ValueJSON {
		data[k] = v
	}
	return data, nil
}

// GetByKey ????
func (s *SettingService) GetByKey(key string) (models.JSON, error) {
	setting, err := s.repo.GetByKey(key)
	if err != nil {
		return nil, err
	}
	if setting == nil {
		return nil, nil
	}
	return setting.ValueJSON, nil
}

// Update ???
func (s *SettingService) Update(key string, value map[string]interface{}) (models.JSON, error) {
	normalized := normalizeSettingValueByKey(key, value)

	setting, err := s.repo.Upsert(key, normalized)
	if err != nil {
		return nil, err
	}
	return setting.ValueJSON, nil
}

// DefaultOrderConfig ???????
func DefaultOrderConfig() OrderConfig {
	return OrderConfig{
		PaymentExpireMinutes: orderPaymentExpireMinutesDefault,
		MaxRefundDays:        orderRefundMaxDaysDefault,
	}
}

// NormalizeOrderConfig ????????
func NormalizeOrderConfig(cfg OrderConfig) OrderConfig {
	if cfg.PaymentExpireMinutes < orderPaymentExpireMinutesMin {
		cfg.PaymentExpireMinutes = orderPaymentExpireMinutesDefault
	}
	if cfg.PaymentExpireMinutes > orderPaymentExpireMinutesMax {
		cfg.PaymentExpireMinutes = orderPaymentExpireMinutesMax
	}
	if cfg.MaxRefundDays < orderRefundMaxDaysMin {
		cfg.MaxRefundDays = orderRefundMaxDaysDefault
	}
	if cfg.MaxRefundDays > orderRefundMaxDaysMax {
		cfg.MaxRefundDays = orderRefundMaxDaysMax
	}
	return cfg
}

// DefaultOrderRefundConfig ?????????
func DefaultOrderRefundConfig() OrderRefundConfig {
	return OrderRefundConfig{
		MaxRefundDays: DefaultOrderConfig().MaxRefundDays,
	}
}

// NormalizeOrderRefundConfig ??????????
func NormalizeOrderRefundConfig(cfg OrderRefundConfig) OrderRefundConfig {
	normalized := NormalizeOrderConfig(OrderConfig{
		PaymentExpireMinutes: DefaultOrderConfig().PaymentExpireMinutes,
		MaxRefundDays:        cfg.MaxRefundDays,
	})
	return OrderRefundConfig{MaxRefundDays: normalized.MaxRefundDays}
}

// orderConfigFromJSON ? JSON map ???????
func orderConfigFromJSON(raw models.JSON, fallback OrderConfig) OrderConfig {
	result := NormalizeOrderConfig(fallback)
	if raw == nil {
		return result
	}
	if parsed, err := parseSettingInt(raw[orderConfigFieldPaymentExpireMinutes]); err == nil {
		result.PaymentExpireMinutes = parsed
	}
	if parsed, err := parseSettingInt(raw[orderConfigFieldMaxRefundDays]); err == nil {
		result.MaxRefundDays = parsed
	}
	return NormalizeOrderConfig(result)
}

// OrderConfigToMap ??????? map ?????
func OrderConfigToMap(cfg OrderConfig) models.JSON {
	normalized := NormalizeOrderConfig(cfg)
	data, err := json.Marshal(normalized)
	if err != nil {
		return models.JSON{}
	}
	var result models.JSON
	_ = json.Unmarshal(data, &result)
	return result
}

func defaultOrderConfigWithFallback(defaultCfg config.OrderConfig, serviceDefault config.OrderConfig, useServiceDefault bool) OrderConfig {
	cfg := DefaultOrderConfig()
	if useServiceDefault {
		if serviceDefault.PaymentExpireMinutes > 0 {
			cfg.PaymentExpireMinutes = serviceDefault.PaymentExpireMinutes
		}
		if serviceDefault.MaxRefundDays >= orderRefundMaxDaysMin && serviceDefault.MaxRefundDays <= orderRefundMaxDaysMax {
			cfg.MaxRefundDays = serviceDefault.MaxRefundDays
		}
	}
	if defaultCfg.PaymentExpireMinutes > 0 {
		cfg.PaymentExpireMinutes = defaultCfg.PaymentExpireMinutes
	}
	// 0 ?????,????????????????(??????? config.yaml ?? 0)?
	if defaultCfg.MaxRefundDays > 0 {
		cfg.MaxRefundDays = defaultCfg.MaxRefundDays
	}
	return NormalizeOrderConfig(cfg)
}

// GetOrderConfig ???????
func (s *SettingService) GetOrderConfig(defaultCfg config.OrderConfig) (OrderConfig, error) {
	var serviceDefault config.OrderConfig
	useServiceDefault := false
	if s != nil {
		serviceDefault = s.defaultOrderConfig
		useServiceDefault = s.hasDefaultOrderConfig
	}
	fallback := defaultOrderConfigWithFallback(defaultCfg, serviceDefault, useServiceDefault)
	if s == nil {
		return fallback, nil
	}
	value, err := s.GetByKey(constants.SettingKeyOrderConfig)
	if err != nil {
		return fallback, err
	}
	return orderConfigFromJSON(value, fallback), nil
}

// GetOrderRefundConfig ?????????
func (s *SettingService) GetOrderRefundConfig() (OrderRefundConfig, error) {
	fallback := DefaultOrderRefundConfig()
	cfg, err := s.GetOrderConfig(config.OrderConfig{})
	if err != nil {
		return fallback, err
	}
	return OrderRefundConfig{MaxRefundDays: cfg.MaxRefundDays}, nil
}

// isOrderRefundWindowExpired ????????????????(?? paid_at,?? created_at)?
func isOrderRefundWindowExpired(order *models.Order, maxRefundDays int, now time.Time) bool {
	normalizedDays := NormalizeOrderRefundConfig(OrderRefundConfig{
		MaxRefundDays: maxRefundDays,
	}).MaxRefundDays
	if order == nil || normalizedDays == 0 {
		return false
	}
	baseAt := order.CreatedAt
	if order.PaidAt != nil && !order.PaidAt.IsZero() {
		baseAt = *order.PaidAt
	}
	if baseAt.IsZero() {
		return false
	}
	deadline := baseAt.AddDate(0, 0, normalizedDays)
	return now.After(deadline)
}

// GetOrderPaymentExpireMinutes ??????????
func (s *SettingService) GetOrderPaymentExpireMinutes(defaultValue int) (int, error) {
	fallback := defaultValue
	if fallback < orderPaymentExpireMinutesMin {
		fallback = orderPaymentExpireMinutesDefault
	}
	if fallback > orderPaymentExpireMinutesMax {
		fallback = orderPaymentExpireMinutesMax
	}
	if s == nil {
		return fallback, nil
	}
	cfg, err := s.GetOrderConfig(config.OrderConfig{
		PaymentExpireMinutes: fallback,
	})
	if err != nil {
		return fallback, err
	}
	return cfg.PaymentExpireMinutes, nil
}

// GetRegistrationEnabled ??????
func (s *SettingService) GetRegistrationEnabled(defaultValue bool) (bool, error) {
	if s == nil {
		return defaultValue, nil
	}
	value, err := s.GetByKey(constants.SettingKeyRegistrationConfig)
	if err != nil {
		return defaultValue, err
	}
	if value == nil {
		return defaultValue, nil
	}
	raw, ok := value[constants.SettingFieldRegistrationEnabled]
	if !ok {
		return defaultValue, nil
	}
	return parseSettingBool(raw), nil
}

// GetEmailVerificationEnabled ????????
func (s *SettingService) GetEmailVerificationEnabled(defaultValue bool) (bool, error) {
	if s == nil {
		return defaultValue, nil
	}
	value, err := s.GetByKey(constants.SettingKeyRegistrationConfig)
	if err != nil {
		return defaultValue, err
	}
	if value == nil {
		return defaultValue, nil
	}
	raw, ok := value[constants.SettingFieldEmailVerificationEnabled]
	if !ok {
		return defaultValue, nil
	}
	return parseSettingBool(raw), nil
}

// GetSiteCurrency ????????
func (s *SettingService) GetSiteCurrency(defaultValue string) (string, error) {
	fallback := normalizeSiteCurrency(defaultValue)
	if s == nil {
		return fallback, nil
	}
	value, err := s.GetByKey(constants.SettingKeySiteConfig)
	if err != nil {
		return fallback, err
	}
	if value == nil {
		return fallback, nil
	}
	raw, ok := value[constants.SettingFieldSiteCurrency]
	if !ok {
		return fallback, nil
	}
	return normalizeSiteCurrency(raw), nil
}

// GetSiteBrand ????????(brand.site_name / brand.site_url)
func (s *SettingService) GetSiteBrand() (SiteBrand, error) {
	if s == nil {
		return SiteBrand{}, nil
	}
	value, err := s.GetByKey(constants.SettingKeySiteConfig)
	if err != nil {
		return SiteBrand{}, err
	}
	if value == nil {
		return SiteBrand{}, nil
	}
	rawBrand, ok := value["brand"]
	if !ok || rawBrand == nil {
		return SiteBrand{}, nil
	}
	brand, ok := rawBrand.(map[string]interface{})
	if !ok || brand == nil {
		return SiteBrand{}, nil
	}
	return SiteBrand{
		SiteName: normalizeSettingText(brand["site_name"]),
		SiteURL:  strings.TrimRight(normalizeSettingText(brand["site_url"]), "/"),
	}, nil
}

// GetWalletOnlyPayment ?????????????
func (s *SettingService) GetWalletOnlyPayment() bool {
	if s == nil {
		return false
	}
	value, err := s.GetByKey(constants.SettingKeyWalletConfig)
	if err != nil || value == nil {
		return false
	}
	raw, ok := value[constants.SettingFieldWalletOnlyPayment]
	if !ok {
		return false
	}
	return parseSettingBool(raw)
}

// GetCallbackRoutes ?????????????????? nil?
func (s *SettingService) GetCallbackRoutes() *CallbackRoutesSetting {
	if s == nil {
		return nil
	}
	value, err := s.GetByKey(constants.SettingKeyCallbackRoutesConfig)
	if err != nil || value == nil {
		return nil
	}
	setting := callbackRoutesSettingFromJSON(value)
	if !setting.HasCustomRoutes() {
		return nil
	}
	return &setting
}

// GetWalletRechargeChannelIDs ?????????????ID??
func (s *SettingService) GetWalletRechargeChannelIDs() []uint {
	if s == nil {
		return nil
	}
	value, err := s.GetByKey(constants.SettingKeyWalletConfig)
	if err != nil || value == nil {
		return nil
	}
	raw, ok := value["recharge_channel_ids"]
	if !ok {
		return nil
	}
	arr, ok := raw.([]interface{})
	if !ok {
		return nil
	}
	result := make([]uint, 0, len(arr))
	for _, item := range arr {
		switch v := item.(type) {
		case float64:
			if v > 0 {
				result = append(result, uint(v))
			}
		case int:
			if v > 0 {
				result = append(result, uint(v))
			}
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
