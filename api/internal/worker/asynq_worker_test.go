package worker

import (
	"context"
	"errors"
	"testing"

	"github.com/dujiao-next/internal/config"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/provider"
	"github.com/dujiao-next/internal/queue"
	"github.com/dujiao-next/internal/repository"
	"github.com/dujiao-next/internal/service"
)

func TestBuildBotNotifyRequestURLReplacesPath(t *testing.T) {
	got, err := buildBotNotifyRequestURL("https://bot.example.com/internal/order-fulfilled", "/internal/wallet-recharge-succeeded")
	if err != nil {
		t.Fatalf("build bot notify request url failed: %v", err)
	}
	want := "https://bot.example.com/internal/wallet-recharge-succeeded"
	if got != want {
		t.Fatalf("request url want %s got %s", want, got)
	}
}

func TestBuildBotNotifyRequestURLReplacesOrderPaidPath(t *testing.T) {
	got, err := buildBotNotifyRequestURL("https://bot.example.com/internal/order-fulfilled", "/internal/order-paid")
	if err != nil {
		t.Fatalf("build bot notify request url failed: %v", err)
	}
	want := "https://bot.example.com/internal/order-paid"
	if got != want {
		t.Fatalf("request url want %s got %s", want, got)
	}
}

func TestBuildOrderFulfillmentEmailPayloadNilOrder(t *testing.T) {
	if got := buildOrderFulfillmentEmailPayload(nil); got != "" {
		t.Fatalf("expected empty payload for nil order, got %q", got)
	}
}

func TestBuildOrderInstructionsEmailText(t *testing.T) {
	t.Run("nil order returns empty", func(t *testing.T) {
		if got := buildOrderInstructionsEmailText(nil, "zh-CN"); got != "" {
			t.Fatalf("expected empty, got %q", got)
		}
	})

	t.Run("locale preferred over fallback", func(t *testing.T) {
		order := &models.Order{
			Items: []models.OrderItem{
				{InstructionsJSON: models.JSON{
					"zh-CN": "<p>????</p>",
					"en-US": "<p>English</p>",
				}},
			},
		}
		if got := buildOrderInstructionsEmailText(order, "en-US"); got != "English" {
			t.Fatalf("want 'English', got %q", got)
		}
	})

	t.Run("falls back to zh-CN when locale missing", func(t *testing.T) {
		order := &models.Order{
			Items: []models.OrderItem{
				{InstructionsJSON: models.JSON{"zh-CN": "fallback"}},
			},
		}
		if got := buildOrderInstructionsEmailText(order, "ja-JP"); got != "fallback" {
			t.Fatalf("want 'fallback', got %q", got)
		}
	})

	t.Run("dedupes identical items and joins distinct", func(t *testing.T) {
		order := &models.Order{
			Items: []models.OrderItem{
				{InstructionsJSON: models.JSON{"zh-CN": "<p>A</p>"}},
				{InstructionsJSON: models.JSON{"zh-CN": "<p>A</p>"}}, // ??,???
				{InstructionsJSON: models.JSON{"zh-CN": "<p>B</p>"}},
			},
		}
		got := buildOrderInstructionsEmailText(order, "zh-CN")
		if got != "A\n\nB" {
			t.Fatalf("want 'A\\n\\nB', got %q", got)
		}
	})

	t.Run("collects from children items", func(t *testing.T) {
		order := &models.Order{
			Children: []models.Order{
				{Items: []models.OrderItem{{InstructionsJSON: models.JSON{"zh-CN": "child1"}}}},
				{Items: []models.OrderItem{{InstructionsJSON: models.JSON{"zh-CN": "child2"}}}},
			},
		}
		got := buildOrderInstructionsEmailText(order, "zh-CN")
		if got != "child1\n\nchild2" {
			t.Fatalf("unexpected: %q", got)
		}
	})

	t.Run("empty instructions yield empty result", func(t *testing.T) {
		order := &models.Order{
			Items: []models.OrderItem{{InstructionsJSON: nil}},
		}
		if got := buildOrderInstructionsEmailText(order, "zh-CN"); got != "" {
			t.Fatalf("expected empty, got %q", got)
		}
	})

	t.Run("strips HTML from instructions", func(t *testing.T) {
		order := &models.Order{
			Items: []models.OrderItem{
				{InstructionsJSON: models.JSON{"zh-CN": "<p>???</p><ul><li>??</li><li>??</li></ul>"}},
			},
		}
		got := buildOrderInstructionsEmailText(order, "zh-CN")
		want := "???\n\n. ??\n. ??"
		if got != want {
			t.Fatalf("want %q, got %q", want, got)
		}
	})
}

func TestBuildOrderFulfillmentEmailPayloadPreferOrderFulfillment(t *testing.T) {
	order := &models.Order{
		Fulfillment: &models.Fulfillment{Payload: "  MAIN-LINE-1\nMAIN-LINE-2  "},
		Children: []models.Order{
			{
				OrderNo:     "CHILD-1",
				Fulfillment: &models.Fulfillment{Payload: "SECRET-1"},
			},
		},
	}

	got := buildOrderFulfillmentEmailPayload(order)
	want := "MAIN-LINE-1\nMAIN-LINE-2"
	if got != want {
		t.Fatalf("unexpected payload, want %q, got %q", want, got)
	}
}

type orderStatusEmailWorkerOrderRepoStub struct {
	repository.OrderRepository
	order *models.Order
	err   error
}

func (s orderStatusEmailWorkerOrderRepoStub) GetByID(_ uint) (*models.Order, error) {
	return s.order, s.err
}

func TestHandleOrderStatusEmailSkipsNonRetryableEmailErrors(t *testing.T) {
	testCases := []struct {
		name         string
		order        *models.Order
		emailConfig  config.EmailConfig
		expectNilErr bool
	}{
		{
			name: "smtp_disabled",
			order: &models.Order{
				ID:          1,
				OrderNo:     "DJ-ORDER-001",
				GuestEmail:  "buyer@example.com",
				GuestLocale: "zh-CN",
				Currency:    "CNY",
			},
			emailConfig:  config.EmailConfig{Enabled: false},
			expectNilErr: true,
		},
		{
			name: "smtp_not_configured",
			order: &models.Order{
				ID:          2,
				OrderNo:     "DJ-ORDER-002",
				GuestEmail:  "buyer@example.com",
				GuestLocale: "zh-CN",
				Currency:    "CNY",
			},
			emailConfig:  config.EmailConfig{Enabled: true},
			expectNilErr: true,
		},
		{
			name: "invalid_receiver_email",
			order: &models.Order{
				ID:          3,
				OrderNo:     "DJ-ORDER-003",
				GuestEmail:  "invalid-email",
				GuestLocale: "zh-CN",
				Currency:    "CNY",
			},
			emailConfig: config.EmailConfig{
				Enabled: true,
				Host:    "127.0.0.1",
				Port:    1,
				From:    "sender@example.com",
			},
			expectNilErr: true,
		},
		{
			name: "generic_send_failure_keeps_retryable_error",
			order: &models.Order{
				ID:          4,
				OrderNo:     "DJ-ORDER-004",
				GuestEmail:  "buyer@example.com",
				GuestLocale: "zh-CN",
				Currency:    "CNY",
			},
			emailConfig: config.EmailConfig{
				Enabled: true,
				Host:    "127.0.0.1",
				Port:    1,
				From:    "sender@example.com",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			task, err := queue.NewOrderStatusEmailTask(queue.OrderStatusEmailPayload{
				OrderID: tc.order.ID,
				Status:  "paid",
			})
			if err != nil {
				t.Fatalf("new order status email task failed: %v", err)
			}

			consumer := &Consumer{
				Container: &provider.Container{
					OrderRepo:    orderStatusEmailWorkerOrderRepoStub{order: tc.order},
					EmailService: service.NewEmailService(&tc.emailConfig),
				},
			}

			err = consumer.handleOrderStatusEmail(context.Background(), task)
			if tc.expectNilErr {
				if err != nil {
					t.Fatalf("expected nil error, got %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected retryable send error, got nil")
			}
			if errors.Is(err, service.ErrEmailServiceDisabled) || errors.Is(err, service.ErrEmailServiceNotConfigured) || errors.Is(err, service.ErrInvalidEmail) {
				t.Fatalf("expected generic retryable error, got %v", err)
			}
		})
	}
}

func TestBuildOrderFulfillmentEmailPayloadFromChildren(t *testing.T) {
	order := &models.Order{
		Children: []models.Order{
			{
				OrderNo:     "DJ-CHILD-01",
				Fulfillment: &models.Fulfillment{Payload: "  SECRET-01  "},
			},
			{
				OrderNo:     "DJ-CHILD-02",
				Fulfillment: nil,
			},
			{
				OrderNo:     "DJ-CHILD-03",
				Fulfillment: &models.Fulfillment{Payload: "    "},
			},
			{
				OrderNo:     "DJ-CHILD-04",
				Fulfillment: &models.Fulfillment{Payload: "SECRET-04-L1\nSECRET-04-L2"},
			},
		},
	}

	got := buildOrderFulfillmentEmailPayload(order)
	want := "[DJ-CHILD-01]\nSECRET-01\n\n[DJ-CHILD-04]\nSECRET-04-L1\nSECRET-04-L2"
	if got != want {
		t.Fatalf("unexpected payload, want %q, got %q", want, got)
	}
}
