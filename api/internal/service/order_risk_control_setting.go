package service

import (
	"encoding/json"
	"net"
	"strings"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
)

// OrderRateLimitConfig ????????
type OrderRateLimitConfig struct {
	Enabled       bool `json:"enabled"`
	WindowSeconds int  `json:"window_seconds"`
	MaxRequests   int  `json:"max_requests"`
	BlockSeconds  int  `json:"block_seconds"`
}

// OrderRiskControlConfig ??????
type OrderRiskControlConfig struct {
	Enabled                       bool                 `json:"enabled"`
	MaxPendingOrdersPerUser       int                  `json:"max_pending_orders_per_user"`
	MaxPendingOrdersPerIP         int                  `json:"max_pending_orders_per_ip"`
	MaxPendingOrdersPerGuestEmail int                  `json:"max_pending_orders_per_guest_email"`
	OrderRateLimit                OrderRateLimitConfig `json:"order_rate_limit"`
	IPBlacklist                   []string             `json:"ip_blacklist"`
	EmailBlacklist                []string             `json:"email_blacklist"`
}

// DefaultOrderRiskControlConfig ??????
func DefaultOrderRiskControlConfig() OrderRiskControlConfig {
	return OrderRiskControlConfig{
		Enabled:                       false,
		MaxPendingOrdersPerUser:       3,
		MaxPendingOrdersPerIP:         5,
		MaxPendingOrdersPerGuestEmail: 2,
		OrderRateLimit: OrderRateLimitConfig{
			Enabled:       false,
			WindowSeconds: 60,
			MaxRequests:   5,
			BlockSeconds:  120,
		},
		IPBlacklist:    []string{},
		EmailBlacklist: []string{},
	}
}

// NormalizeOrderRiskControlConfig ???????
func NormalizeOrderRiskControlConfig(cfg OrderRiskControlConfig) OrderRiskControlConfig {
	if cfg.MaxPendingOrdersPerUser < 0 || cfg.MaxPendingOrdersPerUser > 100 {
		cfg.MaxPendingOrdersPerUser = 3
	}
	if cfg.MaxPendingOrdersPerIP < 0 || cfg.MaxPendingOrdersPerIP > 100 {
		cfg.MaxPendingOrdersPerIP = 5
	}
	if cfg.MaxPendingOrdersPerGuestEmail < 0 || cfg.MaxPendingOrdersPerGuestEmail > 100 {
		cfg.MaxPendingOrdersPerGuestEmail = 2
	}

	if cfg.OrderRateLimit.WindowSeconds < 10 || cfg.OrderRateLimit.WindowSeconds > 3600 {
		cfg.OrderRateLimit.WindowSeconds = 60
	}
	if cfg.OrderRateLimit.MaxRequests < 1 || cfg.OrderRateLimit.MaxRequests > 100 {
		cfg.OrderRateLimit.MaxRequests = 5
	}
	if cfg.OrderRateLimit.BlockSeconds < 0 || cfg.OrderRateLimit.BlockSeconds > 86400 {
		cfg.OrderRateLimit.BlockSeconds = 120
	}

	// ??? IP ???:??????????????
	cleanIPs := make([]string, 0, len(cfg.IPBlacklist))
	for _, entry := range cfg.IPBlacklist {
		entry = trimString(entry)
		if entry == "" {
			continue
		}
		if isValidIPOrCIDR(entry) {
			cleanIPs = append(cleanIPs, entry)
		}
	}
	cfg.IPBlacklist = cleanIPs

	// ????????:???????
	cleanEmails := make([]string, 0, len(cfg.EmailBlacklist))
	for _, email := range cfg.EmailBlacklist {
		email = trimStringToLower(email)
		if email != "" {
			cleanEmails = append(cleanEmails, email)
		}
	}
	cfg.EmailBlacklist = cleanEmails

	return cfg
}

func trimString(s string) string {
	return strings.TrimSpace(s)
}

func trimStringToLower(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

// isValidIPOrCIDR ??????????? IP ??? CIDR ??
func isValidIPOrCIDR(s string) bool {
	if net.ParseIP(s) != nil {
		return true
	}
	_, _, err := net.ParseCIDR(s)
	return err == nil
}

// orderRiskControlConfigFromJSON ? JSON map ??????
func orderRiskControlConfigFromJSON(raw models.JSON, fallback OrderRiskControlConfig) OrderRiskControlConfig {
	result := fallback
	if raw == nil {
		return result
	}
	data, err := json.Marshal(raw)
	if err != nil {
		return result
	}
	_ = json.Unmarshal(data, &result)
	return NormalizeOrderRiskControlConfig(result)
}

// OrderRiskControlConfigToMap ??????? map ????
func OrderRiskControlConfigToMap(cfg OrderRiskControlConfig) models.JSON {
	normalized := NormalizeOrderRiskControlConfig(cfg)
	data, err := json.Marshal(normalized)
	if err != nil {
		return models.JSON{}
	}
	var result models.JSON
	_ = json.Unmarshal(data, &result)
	return result
}

// GetOrderRiskControlConfig ??????
func (s *SettingService) GetOrderRiskControlConfig() (OrderRiskControlConfig, error) {
	fallback := DefaultOrderRiskControlConfig()
	if s == nil {
		return fallback, nil
	}
	value, err := s.GetByKey(constants.SettingKeyOrderRiskControlConfig)
	if err != nil {
		return fallback, err
	}
	return orderRiskControlConfigFromJSON(value, fallback), nil
}
