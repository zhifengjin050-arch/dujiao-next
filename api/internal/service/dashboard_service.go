package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dujiao-next/internal/cache"
	"github.com/dujiao-next/internal/repository"
)

const (
	dashboardCacheTTL      = 45 * time.Second
	dashboardCustomMaxDays = 90
)

// DashboardService ?????
// ??:?????????????
type DashboardService struct {
	repo           repository.DashboardRepository
	settingService *SettingService
}

// NewDashboardService ???????
func NewDashboardService(repo repository.DashboardRepository, settingService *SettingService) *DashboardService {
	return &DashboardService{repo: repo, settingService: settingService}
}

// DashboardQueryInput ???????
type DashboardQueryInput struct {
	Range        string
	From         *time.Time
	To           *time.Time
	Timezone     string
	ForceRefresh bool
}

// DashboardOverviewResponse ???????
type DashboardOverviewResponse struct {
	Range    string               `json:"range"`
	From     string               `json:"from"`
	To       string               `json:"to"`
	Timezone string               `json:"timezone"`
	Currency string               `json:"currency,omitempty"`
	KPI      DashboardKPI         `json:"kpi"`
	Funnel   DashboardFunnel      `json:"funnel"`
	Alerts   []DashboardAlertItem `json:"alerts"`
}

// DashboardKPI ???????
type DashboardKPI struct {
	OrdersTotal          int64  `json:"orders_total"`
	PaidOrders           int64  `json:"paid_orders"`
	CompletedOrders      int64  `json:"completed_orders"`
	PendingPaymentOrders int64  `json:"pending_payment_orders"`
	ProcessingOrders     int64  `json:"processing_orders"`
	GMVPaid              string `json:"gmv_paid"`
	TotalCost            string `json:"total_cost"`
	TotalProfit          string `json:"total_profit"`
	ProfitMargin         string `json:"profit_margin"`
	PaymentsTotal        int64  `json:"payments_total"`
	PaymentsSuccess      int64  `json:"payments_success"`
	PaymentsFailed       int64  `json:"payments_failed"`
	PaymentSuccessRate   string `json:"payment_success_rate"`
	NewUsers             int64  `json:"new_users"`
	ActiveProducts       int64  `json:"active_products"`
	OutOfStockProducts   int64  `json:"out_of_stock_products"`
	LowStockProducts     int64  `json:"low_stock_products"`
	OutOfStockSKUs       int64  `json:"out_of_stock_skus"`
	LowStockSKUs         int64  `json:"low_stock_skus"`
	AutoAvailableSecrets int64  `json:"auto_available_secrets"`
	ManualAvailableUnits int64  `json:"manual_available_units"`
	TotalUserBalance     string `json:"total_user_balance"`
}

// DashboardFunnel ???????
type DashboardFunnel struct {
	OrdersCreated         int64  `json:"orders_created"`
	PaymentsCreated       int64  `json:"payments_created"`
	PaymentsSuccess       int64  `json:"payments_success"`
	OrdersPaid            int64  `json:"orders_paid"`
	OrdersCompleted       int64  `json:"orders_completed"`
	PaymentConversionRate string `json:"payment_conversion_rate"`
	CompletionRate        string `json:"completion_rate"`
}

// DashboardAlertItem ??????
type DashboardAlertItem struct {
	Type  string `json:"type"`
	Level string `json:"level"`
	Value int64  `json:"value"`
}

// DashboardTrendResponse ???????
type DashboardTrendResponse struct {
	Range    string                `json:"range"`
	From     string                `json:"from"`
	To       string                `json:"to"`
	Timezone string                `json:"timezone"`
	Points   []DashboardTrendPoint `json:"points"`
}

// DashboardTrendPoint ???
type DashboardTrendPoint struct {
	Date            string `json:"date"`
	OrdersTotal     int64  `json:"orders_total"`
	OrdersPaid      int64  `json:"orders_paid"`
	PaymentsSuccess int64  `json:"payments_success"`
	PaymentsFailed  int64  `json:"payments_failed"`
	GMVPaid         string `json:"gmv_paid"`
	Profit          string `json:"profit"`
}

// DashboardRankingsResponse ????????
type DashboardRankingsResponse struct {
	Range       string                    `json:"range"`
	From        string                    `json:"from"`
	To          string                    `json:"to"`
	Timezone    string                    `json:"timezone"`
	TopProducts []DashboardProductRanking `json:"top_products"`
	TopChannels []DashboardChannelRanking `json:"top_channels"`
}

// DashboardProductRanking ?????
type DashboardProductRanking struct {
	ProductID  uint   `json:"product_id"`
	Title      string `json:"title"`
	PaidOrders int64  `json:"paid_orders"`
	Quantity   int64  `json:"quantity"`
	PaidAmount string `json:"paid_amount"`
	TotalCost  string `json:"total_cost"`
	Profit     string `json:"profit"`
}

// DashboardChannelRanking ?????
type DashboardChannelRanking struct {
	ChannelID     uint   `json:"channel_id"`
	ChannelName   string `json:"channel_name"`
	ProviderType  string `json:"provider_type"`
	ChannelType   string `json:"channel_type"`
	SuccessCount  int64  `json:"success_count"`
	FailedCount   int64  `json:"failed_count"`
	SuccessAmount string `json:"success_amount"`
	SuccessRate   string `json:"success_rate"`
}

type dashboardWindow struct {
	rangeKey string
	startAt  time.Time
	endAt    time.Time
	timezone string
}

// GetOverview ???????
func (s *DashboardService) GetOverview(ctx context.Context, input DashboardQueryInput) (*DashboardOverviewResponse, error) {
	if s == nil || s.repo == nil {
		return &DashboardOverviewResponse{}, nil
	}

	window, err := resolveDashboardWindow(input, time.Now())
	if err != nil {
		return nil, err
	}

	setting := s.loadDashboardSetting()

	cacheKey := fmt.Sprintf("dashboard:overview:%s:%d:%d:%s:%d:%d:%d:%d",
		window.rangeKey,
		window.startAt.Unix(),
		window.endAt.Unix(),
		window.timezone,
		setting.Alert.LowStockThreshold,
		setting.Alert.OutOfStockProductsThreshold,
		setting.Alert.PendingPaymentOrdersThreshold,
		setting.Alert.PaymentsFailedThreshold,
	)
	if !input.ForceRefresh {
		var cached DashboardOverviewResponse
		hit, cacheErr := cache.GetJSON(ctx, cacheKey, &cached)
		if cacheErr == nil && hit {
			return &cached, nil
		}
	}

	overview, err := s.repo.GetOverview(window.startAt, window.endAt)
	if err != nil {
		return nil, err
	}
	profitOverview, err := s.repo.GetProfitOverview(window.startAt, window.endAt)
	if err != nil {
		return nil, err
	}
	stockStats, err := s.repo.GetStockStats(setting.Alert.LowStockThreshold)
	if err != nil {
		return nil, err
	}
	totalUserBalance, err := s.repo.GetTotalUserBalance()
	if err != nil {
		return nil, err
	}

	totalProfit := profitOverview.TotalRevenue - profitOverview.TotalCost
	profitMargin := 0.0
	if profitOverview.TotalRevenue > 0 {
		profitMargin = totalProfit / profitOverview.TotalRevenue * 100
	}

	paymentSuccessRate := 0.0
	if overview.PaymentsTotal > 0 {
		paymentSuccessRate = float64(overview.PaymentsSuccess) / float64(overview.PaymentsTotal) * 100
	}

	paymentConversionRate := 0.0
	if overview.OrdersTotal > 0 {
		paymentConversionRate = float64(overview.PaidOrders) / float64(overview.OrdersTotal) * 100
	}

	completionRate := 0.0
	if overview.PaidOrders > 0 {
		completionRate = float64(overview.CompletedOrders) / float64(overview.PaidOrders) * 100
	}

	response := &DashboardOverviewResponse{
		Range:    window.rangeKey,
		From:     window.startAt.Format(time.RFC3339),
		To:       window.endAt.Add(-time.Second).Format(time.RFC3339),
		Timezone: window.timezone,
		Currency: strings.ToUpper(strings.TrimSpace(overview.Currency)),
		KPI: DashboardKPI{
			OrdersTotal:          overview.OrdersTotal,
			PaidOrders:           overview.PaidOrders,
			CompletedOrders:      overview.CompletedOrders,
			PendingPaymentOrders: overview.PendingPaymentOrders,
			ProcessingOrders:     overview.ProcessingOrders,
			GMVPaid:              formatMoneyValue(overview.GMVPaid),
			TotalCost:            formatMoneyValue(profitOverview.TotalCost),
			TotalProfit:          formatMoneyValue(totalProfit),
			ProfitMargin:         formatPercentValue(profitMargin),
			PaymentsTotal:        overview.PaymentsTotal,
			PaymentsSuccess:      overview.PaymentsSuccess,
			PaymentsFailed:       overview.PaymentsFailed,
			PaymentSuccessRate:   formatPercentValue(paymentSuccessRate),
			NewUsers:             overview.NewUsers,
			ActiveProducts:       overview.ActiveProducts,
			OutOfStockProducts:   stockStats.OutOfStockProducts,
			LowStockProducts:     stockStats.LowStockProducts,
			OutOfStockSKUs:       stockStats.OutOfStockSKUs,
			LowStockSKUs:         stockStats.LowStockSKUs,
			AutoAvailableSecrets: stockStats.AutoAvailableSecrets,
			ManualAvailableUnits: stockStats.ManualAvailableUnits,
			TotalUserBalance:     formatMoneyValue(totalUserBalance),
		},
		Funnel: DashboardFunnel{
			OrdersCreated:         overview.OrdersTotal,
			PaymentsCreated:       overview.PaymentsTotal,
			PaymentsSuccess:       overview.PaymentsSuccess,
			OrdersPaid:            overview.PaidOrders,
			OrdersCompleted:       overview.CompletedOrders,
			PaymentConversionRate: formatPercentValue(paymentConversionRate),
			CompletionRate:        formatPercentValue(completionRate),
		},
		Alerts: buildDashboardAlerts(overview, stockStats, setting.Alert),
	}

	_ = cache.SetJSON(ctx, cacheKey, response, dashboardCacheTTL)
	return response, nil
}

// LoadDashboardAlertSetting ?????????
func (s *DashboardService) LoadDashboardAlertSetting() DashboardAlertSetting {
	return s.loadDashboardSetting().Alert
}

// GetInventoryAlertItems ????????
func (s *DashboardService) GetInventoryAlertItems(_ context.Context, lowStockThreshold int64) ([]repository.DashboardInventoryAlertRow, error) {
	if s == nil || s.repo == nil {
		return []repository.DashboardInventoryAlertRow{}, nil
	}
	return s.repo.GetInventoryAlertItems(lowStockThreshold)
}

// GetTrends ???????
func (s *DashboardService) GetTrends(ctx context.Context, input DashboardQueryInput) (*DashboardTrendResponse, error) {
	if s == nil || s.repo == nil {
		return &DashboardTrendResponse{}, nil
	}

	window, err := resolveDashboardWindow(input, time.Now())
	if err != nil {
		return nil, err
	}

	cacheKey := fmt.Sprintf("dashboard:trends:%s:%d:%d:%s", window.rangeKey, window.startAt.Unix(), window.endAt.Unix(), window.timezone)
	if !input.ForceRefresh {
		var cached DashboardTrendResponse
		hit, cacheErr := cache.GetJSON(ctx, cacheKey, &cached)
		if cacheErr == nil && hit {
			return &cached, nil
		}
	}

	orderRows, err := s.repo.GetOrderTrends(window.startAt, window.endAt)
	if err != nil {
		return nil, err
	}
	paymentRows, err := s.repo.GetPaymentTrends(window.startAt, window.endAt)
	if err != nil {
		return nil, err
	}
	profitRows, err := s.repo.GetProfitTrends(window.startAt, window.endAt)
	if err != nil {
		return nil, err
	}

	orderMap := make(map[string]repository.DashboardOrderTrendRow, len(orderRows))
	for _, item := range orderRows {
		orderMap[item.Day] = item
	}
	paymentMap := make(map[string]repository.DashboardPaymentTrendRow, len(paymentRows))
	for _, item := range paymentRows {
		paymentMap[item.Day] = item
	}
	profitMap := make(map[string]repository.DashboardProfitTrendRow, len(profitRows))
	for _, item := range profitRows {
		profitMap[item.Day] = item
	}

	points := make([]DashboardTrendPoint, 0)
	for cursor := time.Date(window.startAt.Year(), window.startAt.Month(), window.startAt.Day(), 0, 0, 0, 0, window.startAt.Location()); cursor.Before(window.endAt); cursor = cursor.AddDate(0, 0, 1) {
		day := cursor.Format("2006-01-02")
		orderItem := orderMap[day]
		paymentItem := paymentMap[day]
		profitItem := profitMap[day]
		dayProfit := profitItem.Revenue - profitItem.Cost
		points = append(points, DashboardTrendPoint{
			Date:            day,
			OrdersTotal:     orderItem.OrdersTotal,
			OrdersPaid:      orderItem.OrdersPaid,
			PaymentsSuccess: paymentItem.PaymentsSuccess,
			PaymentsFailed:  paymentItem.PaymentsFailed,
			GMVPaid:         formatMoneyValue(paymentItem.GMVPaid),
			Profit:          formatMoneyValue(dayProfit),
		})
	}

	response := &DashboardTrendResponse{
		Range:    window.rangeKey,
		From:     window.startAt.Format(time.RFC3339),
		To:       window.endAt.Add(-time.Second).Format(time.RFC3339),
		Timezone: window.timezone,
		Points:   points,
	}

	_ = cache.SetJSON(ctx, cacheKey, response, dashboardCacheTTL)
	return response, nil
}

// GetRankings ????????
func (s *DashboardService) GetRankings(ctx context.Context, input DashboardQueryInput) (*DashboardRankingsResponse, error) {
	if s == nil || s.repo == nil {
		return &DashboardRankingsResponse{}, nil
	}

	window, err := resolveDashboardWindow(input, time.Now())
	if err != nil {
		return nil, err
	}

	setting := s.loadDashboardSetting()

	cacheKey := fmt.Sprintf("dashboard:rankings:%s:%d:%d:%s:%d:%d",
		window.rangeKey,
		window.startAt.Unix(),
		window.endAt.Unix(),
		window.timezone,
		setting.Ranking.TopProductsLimit,
		setting.Ranking.TopChannelsLimit,
	)
	if !input.ForceRefresh {
		var cached DashboardRankingsResponse
		hit, cacheErr := cache.GetJSON(ctx, cacheKey, &cached)
		if cacheErr == nil && hit {
			return &cached, nil
		}
	}

	productRows, err := s.repo.GetTopProducts(window.startAt, window.endAt, setting.Ranking.TopProductsLimit)
	if err != nil {
		return nil, err
	}
	channelRows, err := s.repo.GetTopChannels(window.startAt, window.endAt, setting.Ranking.TopChannelsLimit)
	if err != nil {
		return nil, err
	}

	products := make([]DashboardProductRanking, 0, len(productRows))
	for _, item := range productRows {
		title := strings.TrimSpace(item.Title)
		if title == "" {
			title = "-"
		}
		products = append(products, DashboardProductRanking{
			ProductID:  item.ProductID,
			Title:      title,
			PaidOrders: item.PaidOrders,
			Quantity:   item.Quantity,
			PaidAmount: formatMoneyValue(item.PaidAmount),
			TotalCost:  formatMoneyValue(item.TotalCost),
			Profit:     formatMoneyValue(item.PaidAmount - item.TotalCost),
		})
	}

	channels := make([]DashboardChannelRanking, 0, len(channelRows))
	for _, item := range channelRows {
		total := item.SuccessCount + item.FailedCount
		rate := 0.0
		if total > 0 {
			rate = float64(item.SuccessCount) / float64(total) * 100
		}
		channels = append(channels, DashboardChannelRanking{
			ChannelID:     item.ChannelID,
			ChannelName:   strings.TrimSpace(item.ChannelName),
			ProviderType:  strings.TrimSpace(item.ProviderType),
			ChannelType:   strings.TrimSpace(item.ChannelType),
			SuccessCount:  item.SuccessCount,
			FailedCount:   item.FailedCount,
			SuccessAmount: formatMoneyValue(item.SuccessAmount),
			SuccessRate:   formatPercentValue(rate),
		})
	}

	response := &DashboardRankingsResponse{
		Range:       window.rangeKey,
		From:        window.startAt.Format(time.RFC3339),
		To:          window.endAt.Add(-time.Second).Format(time.RFC3339),
		Timezone:    window.timezone,
		TopProducts: products,
		TopChannels: channels,
	}

	_ = cache.SetJSON(ctx, cacheKey, response, dashboardCacheTTL)
	return response, nil
}

func (s *DashboardService) loadDashboardSetting() DashboardSetting {
	fallback := DashboardDefaultSetting()
	if s == nil || s.settingService == nil {
		return fallback
	}
	setting, err := s.settingService.GetDashboardSetting()
	if err != nil {
		return fallback
	}
	return NormalizeDashboardSetting(setting)
}

func resolveDashboardWindow(input DashboardQueryInput, now time.Time) (dashboardWindow, error) {
	rangeKey := strings.ToLower(strings.TrimSpace(input.Range))
	if rangeKey == "" {
		rangeKey = "7d"
	}

	timezone := strings.TrimSpace(input.Timezone)
	location := time.Local
	if timezone != "" {
		if parsed, err := time.LoadLocation(timezone); err == nil {
			location = parsed
		} else {
			timezone = ""
		}
	}
	if timezone == "" {
		timezone = location.String()
	}

	localNow := now.In(location)
	todayStart := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, location)
	window := dashboardWindow{rangeKey: rangeKey, timezone: timezone}

	switch rangeKey {
	case "today":
		window.startAt = todayStart
		window.endAt = todayStart.AddDate(0, 0, 1)
	case "7d":
		window.startAt = todayStart.AddDate(0, 0, -6)
		window.endAt = todayStart.AddDate(0, 0, 1)
	case "30d":
		window.startAt = todayStart.AddDate(0, 0, -29)
		window.endAt = todayStart.AddDate(0, 0, 1)
	case "custom":
		if input.From == nil || input.To == nil {
			return dashboardWindow{}, ErrDashboardRangeInvalid
		}
		startAt := input.From.In(location)
		endAt := input.To.In(location)
		if endAt.Before(startAt) {
			return dashboardWindow{}, ErrDashboardRangeInvalid
		}
		if endAt.Sub(startAt) > time.Hour*24*dashboardCustomMaxDays {
			return dashboardWindow{}, ErrDashboardRangeInvalid
		}
		window.startAt = startAt
		window.endAt = endAt.Add(time.Second)
	default:
		return dashboardWindow{}, ErrDashboardRangeInvalid
	}

	if !window.endAt.After(window.startAt) {
		return dashboardWindow{}, ErrDashboardRangeInvalid
	}
	return window, nil
}

func formatMoneyValue(value float64) string {
	return fmt.Sprintf("%.2f", value)
}

func formatPercentValue(value float64) string {
	return fmt.Sprintf("%.2f", value)
}

func buildDashboardAlerts(overview repository.DashboardOverviewRow, stockStats repository.DashboardStockStatsRow, alertSetting DashboardAlertSetting) []DashboardAlertItem {
	alerts := make([]DashboardAlertItem, 0, 4)
	if stockStats.OutOfStockProducts >= alertSetting.OutOfStockProductsThreshold {
		alerts = append(alerts, DashboardAlertItem{Type: "out_of_stock_products", Level: "error", Value: stockStats.OutOfStockProducts})
	}
	if stockStats.LowStockProducts > 0 {
		alerts = append(alerts, DashboardAlertItem{Type: "low_stock_products", Level: "warning", Value: stockStats.LowStockProducts})
	}
	if overview.PendingPaymentOrders >= alertSetting.PendingPaymentOrdersThreshold {
		alerts = append(alerts, DashboardAlertItem{Type: "pending_payment_orders", Level: "warning", Value: overview.PendingPaymentOrders})
	}
	if overview.PaymentsFailed >= alertSetting.PaymentsFailedThreshold {
		alerts = append(alerts, DashboardAlertItem{Type: "payments_failed", Level: "warning", Value: overview.PaymentsFailed})
	}
	return alerts
}
