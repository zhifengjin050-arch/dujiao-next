package service

import (
	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
)

// DashboardAlertSetting ?????????
type DashboardAlertSetting struct {
	LowStockThreshold             int64 `json:"low_stock_threshold"`
	OutOfStockProductsThreshold   int64 `json:"out_of_stock_products_threshold"`
	PendingPaymentOrdersThreshold int64 `json:"pending_payment_orders_threshold"`
	PaymentsFailedThreshold       int64 `json:"payments_failed_threshold"`
}

// DashboardRankingSetting ?????????
type DashboardRankingSetting struct {
	TopProductsLimit int `json:"top_products_limit"`
	TopChannelsLimit int `json:"top_channels_limit"`
}

// DashboardSetting ?????
type DashboardSetting struct {
	Alert   DashboardAlertSetting   `json:"alert"`
	Ranking DashboardRankingSetting `json:"ranking"`
}

// DashboardDefaultSetting ???????
func DashboardDefaultSetting() DashboardSetting {
	return NormalizeDashboardSetting(DashboardSetting{
		Alert: DashboardAlertSetting{
			LowStockThreshold:             5,
			OutOfStockProductsThreshold:   1,
			PendingPaymentOrdersThreshold: 20,
			PaymentsFailedThreshold:       10,
		},
		Ranking: DashboardRankingSetting{
			TopProductsLimit: 5,
			TopChannelsLimit: 5,
		},
	})
}

// NormalizeDashboardSetting ????????
func NormalizeDashboardSetting(setting DashboardSetting) DashboardSetting {
	if setting.Alert.LowStockThreshold < 1 || setting.Alert.LowStockThreshold > 500 {
		setting.Alert.LowStockThreshold = 5
	}
	if setting.Alert.OutOfStockProductsThreshold < 1 || setting.Alert.OutOfStockProductsThreshold > 10000 {
		setting.Alert.OutOfStockProductsThreshold = 1
	}
	if setting.Alert.PendingPaymentOrdersThreshold < 1 || setting.Alert.PendingPaymentOrdersThreshold > 100000 {
		setting.Alert.PendingPaymentOrdersThreshold = 20
	}
	if setting.Alert.PaymentsFailedThreshold < 1 || setting.Alert.PaymentsFailedThreshold > 100000 {
		setting.Alert.PaymentsFailedThreshold = 10
	}

	if setting.Ranking.TopProductsLimit < 1 || setting.Ranking.TopProductsLimit > 20 {
		setting.Ranking.TopProductsLimit = 5
	}
	if setting.Ranking.TopChannelsLimit < 1 || setting.Ranking.TopChannelsLimit > 20 {
		setting.Ranking.TopChannelsLimit = 5
	}

	return setting
}

// DashboardSettingToMap ???????????????
func DashboardSettingToMap(setting DashboardSetting) map[string]interface{} {
	normalized := NormalizeDashboardSetting(setting)
	return map[string]interface{}{
		"alert": map[string]interface{}{
			"low_stock_threshold":              normalized.Alert.LowStockThreshold,
			"out_of_stock_products_threshold":  normalized.Alert.OutOfStockProductsThreshold,
			"pending_payment_orders_threshold": normalized.Alert.PendingPaymentOrdersThreshold,
			"payments_failed_threshold":        normalized.Alert.PaymentsFailedThreshold,
		},
		"ranking": map[string]interface{}{
			"top_products_limit": normalized.Ranking.TopProductsLimit,
			"top_channels_limit": normalized.Ranking.TopChannelsLimit,
		},
	}
}

func dashboardSettingFromJSON(raw models.JSON, fallback DashboardSetting) DashboardSetting {
	result := fallback

	alertRaw, ok := raw["alert"].(map[string]interface{})
	if ok {
		if value, exists := alertRaw["low_stock_threshold"]; exists {
			if parsed, err := parseSettingInt(value); err == nil {
				result.Alert.LowStockThreshold = int64(parsed)
			}
		}
		if value, exists := alertRaw["out_of_stock_products_threshold"]; exists {
			if parsed, err := parseSettingInt(value); err == nil {
				result.Alert.OutOfStockProductsThreshold = int64(parsed)
			}
		}
		if value, exists := alertRaw["pending_payment_orders_threshold"]; exists {
			if parsed, err := parseSettingInt(value); err == nil {
				result.Alert.PendingPaymentOrdersThreshold = int64(parsed)
			}
		}
		if value, exists := alertRaw["payments_failed_threshold"]; exists {
			if parsed, err := parseSettingInt(value); err == nil {
				result.Alert.PaymentsFailedThreshold = int64(parsed)
			}
		}
	}

	rankingRaw, ok := raw["ranking"].(map[string]interface{})
	if ok {
		if value, exists := rankingRaw["top_products_limit"]; exists {
			if parsed, err := parseSettingInt(value); err == nil {
				result.Ranking.TopProductsLimit = parsed
			}
		}
		if value, exists := rankingRaw["top_channels_limit"]; exists {
			if parsed, err := parseSettingInt(value); err == nil {
				result.Ranking.TopChannelsLimit = parsed
			}
		}
	}

	return NormalizeDashboardSetting(result)
}

// GetDashboardSetting ???????(?? settings,??????)
func (s *SettingService) GetDashboardSetting() (DashboardSetting, error) {
	fallback := DashboardDefaultSetting()
	if s == nil {
		return fallback, nil
	}
	value, err := s.GetByKey(constants.SettingKeyDashboardConfig)
	if err != nil {
		return fallback, err
	}
	if value == nil {
		return fallback, nil
	}
	return dashboardSettingFromJSON(value, fallback), nil
}

// GetDashboardLowStockThreshold ???????(?????????)
func (s *SettingService) GetDashboardLowStockThreshold() int {
	defaultThreshold := int(DashboardDefaultSetting().Alert.LowStockThreshold)
	if s == nil {
		return defaultThreshold
	}

	setting, err := s.GetDashboardSetting()
	if err != nil {
		return defaultThreshold
	}
	return int(setting.Alert.LowStockThreshold)
}
