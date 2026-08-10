package xunhupay

import (
	"fmt"
	"strings"

	"github.com/dujiao-next/internal/payment/common"
)

var (
	ErrConfigInvalid = fmt.Errorf("xunhupay config invalid")
)

// Config ??????
// ??:???????? payment_channels.config_json
// ???????? JSON ???
type Config struct {
	common.ExchangeRateConfig
	AppID       string `json:"app_id"`
	AppSecret   string `json:"app_secret"`
	GatewayURL  string `json:"gateway_url"`
	QueryURL    string `json:"query_url"`
	NotifyURL   string `json:"notify_url"`
	ReturnURL   string `json:"return_url"`
	CallbackURL string `json:"callback_url"`
	WapName     string `json:"wap_name"`
	Title       string `json:"title"`
}

// ParseConfig ????
func ParseConfig(raw map[string]interface{}) (*Config, error) {
	return common.ParseConfig[Config](raw, ErrConfigInvalid)
}

// Normalize ?????
func (c *Config) Normalize() {
	c.GatewayURL = strings.TrimSpace(c.GatewayURL)
	c.QueryURL = strings.TrimSpace(c.QueryURL)
	c.NotifyURL = strings.TrimSpace(c.NotifyURL)
	c.ReturnURL = strings.TrimSpace(c.ReturnURL)
	c.CallbackURL = strings.TrimSpace(c.CallbackURL)
	c.WapName = strings.TrimSpace(c.WapName)
	c.AppID = strings.TrimSpace(c.AppID)
	c.AppSecret = strings.TrimSpace(c.AppSecret)
	c.ExchangeRateConfig.NormalizeExchangeRate()
	if c.GatewayURL == "" {
		c.GatewayURL = "https://api.xunhupay.com/payment/do.html"
	}
	if c.QueryURL == "" {
		c.QueryURL = "https://api.xunhupay.com/payment/query.html"
	}
	if c.WapName == "" {
		c.WapName = "??????"
	}
}

// ValidateConfig ???????
func ValidateConfig(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("%w: config is nil", ErrConfigInvalid)
	}
	if strings.TrimSpace(cfg.AppID) == "" {
		return fmt.Errorf("%w: app_id is required", ErrConfigInvalid)
	}
	if strings.TrimSpace(cfg.AppSecret) == "" {
		return fmt.Errorf("%w: app_secret is required", ErrConfigInvalid)
	}
	if strings.TrimSpace(cfg.GatewayURL) == "" {
		return fmt.Errorf("%w: gateway_url is required", ErrConfigInvalid)
	}
	if strings.TrimSpace(cfg.NotifyURL) == "" {
		return fmt.Errorf("%w: notify_url is required", ErrConfigInvalid)
	}
	if strings.TrimSpace(cfg.ReturnURL) == "" {
		return fmt.Errorf("%w: return_url is required", ErrConfigInvalid)
	}
	return nil
}
