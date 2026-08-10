package service

import (
	"context"
	"testing"
	"time"

	"github.com/dujiao-next/internal/repository"
)

type dashboardServiceRepoStub struct {
	overview repository.DashboardOverviewRow
	stock    repository.DashboardStockStatsRow
}

func (s dashboardServiceRepoStub) GetOverview(startAt, endAt time.Time) (repository.DashboardOverviewRow, error) {
	return s.overview, nil
}

func (s dashboardServiceRepoStub) GetOrderTrends(startAt, endAt time.Time) ([]repository.DashboardOrderTrendRow, error) {
	return []repository.DashboardOrderTrendRow{}, nil
}

func (s dashboardServiceRepoStub) GetPaymentTrends(startAt, endAt time.Time) ([]repository.DashboardPaymentTrendRow, error) {
	return []repository.DashboardPaymentTrendRow{}, nil
}

func (s dashboardServiceRepoStub) GetStockStats(lowStockThreshold int64) (repository.DashboardStockStatsRow, error) {
	return s.stock, nil
}

func (s dashboardServiceRepoStub) GetInventoryAlertItems(lowStockThreshold int64) ([]repository.DashboardInventoryAlertRow, error) {
	return []repository.DashboardInventoryAlertRow{}, nil
}

func (s dashboardServiceRepoStub) GetTopProducts(startAt, endAt time.Time, limit int) ([]repository.DashboardProductRankingRow, error) {
	return []repository.DashboardProductRankingRow{}, nil
}

func (s dashboardServiceRepoStub) GetProfitOverview(startAt, endAt time.Time) (repository.DashboardProfitOverviewRow, error) {
	return repository.DashboardProfitOverviewRow{}, nil
}

func (s dashboardServiceRepoStub) GetProfitTrends(startAt, endAt time.Time) ([]repository.DashboardProfitTrendRow, error) {
	return []repository.DashboardProfitTrendRow{}, nil
}

func (s dashboardServiceRepoStub) GetTopChannels(startAt, endAt time.Time, limit int) ([]repository.DashboardChannelRankingRow, error) {
	return []repository.DashboardChannelRankingRow{}, nil
}

func (s dashboardServiceRepoStub) GetTotalUserBalance() (float64, error) {
	return 0, nil
}

func TestDashboardOverviewUsesPaidOrdersForPaymentConversionRate(t *testing.T) {
	service := NewDashboardService(dashboardServiceRepoStub{
		overview: repository.DashboardOverviewRow{
			OrdersTotal:     10,
			PaidOrders:      6,
			CompletedOrders: 3,
			PaymentsTotal:   5,
			PaymentsSuccess: 4,
			Currency:        "cny",
			GMVPaid:         120,
		},
		stock: repository.DashboardStockStatsRow{},
	}, nil)

	response, err := service.GetOverview(context.Background(), DashboardQueryInput{
		Range:    "today",
		Timezone: "Asia/Shanghai",
	})
	if err != nil {
		t.Fatalf("get overview failed: %v", err)
	}
	if response.Currency != "CNY" {
		t.Fatalf("currency want CNY got %s", response.Currency)
	}
	if response.Funnel.PaymentConversionRate != "60.00" {
		t.Fatalf("payment conversion rate want 60.00 got %s", response.Funnel.PaymentConversionRate)
	}
	if response.KPI.PaymentSuccessRate != "80.00" {
		t.Fatalf("payment success rate want 80.00 got %s", response.KPI.PaymentSuccessRate)
	}
}

func TestDashboardOverviewBuildsInventoryAlertsFromStockStats(t *testing.T) {
	service := NewDashboardService(dashboardServiceRepoStub{
		overview: repository.DashboardOverviewRow{
			PendingPaymentOrders: 25,
			PaymentsFailed:       12,
		},
		stock: repository.DashboardStockStatsRow{
			OutOfStockProducts: 2,
			LowStockProducts:   1,
		},
	}, nil)

	response, err := service.GetOverview(context.Background(), DashboardQueryInput{
		Range:    "today",
		Timezone: "Asia/Shanghai",
	})
	if err != nil {
		t.Fatalf("get overview failed: %v", err)
	}
	if len(response.Alerts) != 4 {
		t.Fatalf("alerts len want 4 got %d", len(response.Alerts))
	}
	if response.Alerts[0].Type != "out_of_stock_products" || response.Alerts[0].Value != 2 {
		t.Fatalf("unexpected first alert: %+v", response.Alerts[0])
	}
}

var _ repository.DashboardRepository = dashboardServiceRepoStub{}
