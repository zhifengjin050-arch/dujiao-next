package service

import (
	"errors"
	"testing"

	"github.com/dujiao-next/internal/config"
	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/queue"
	"github.com/dujiao-next/internal/repository"
)

type orderStatusEmailOrderRepoStub struct {
	repository.OrderRepository
	receiver string
	err      error
}

func (s orderStatusEmailOrderRepoStub) ResolveReceiverEmailByOrderID(_ uint) (string, error) {
	return s.receiver, s.err
}

func TestEnqueueOrderStatusEmailTaskIfEligibleSkipTelegramPlaceholder(t *testing.T) {
	queueClient, err := queue.NewClient(nil)
	if err != nil {
		t.Fatalf("new queue client failed: %v", err)
	}
	t.Cleanup(func() {
		_ = queueClient.Close()
	})

	skipped, err := enqueueOrderStatusEmailTaskIfEligible(
		orderStatusEmailOrderRepoStub{receiver: "telegram_123@login.local"},
		queueClient,
		nil,
		config.EmailConfig{},
		101,
		"paid",
	)
	if err != nil {
		t.Fatalf("enqueue helper returned error: %v", err)
	}
	if !skipped {
		t.Fatalf("expected task skipped for telegram placeholder email")
	}
}

func TestEnqueueOrderStatusEmailTaskIfEligibleSkipEmptyReceiver(t *testing.T) {
	queueClient, err := queue.NewClient(nil)
	if err != nil {
		t.Fatalf("new queue client failed: %v", err)
	}
	t.Cleanup(func() {
		_ = queueClient.Close()
	})

	skipped, err := enqueueOrderStatusEmailTaskIfEligible(
		orderStatusEmailOrderRepoStub{receiver: "   "},
		queueClient,
		nil,
		config.EmailConfig{},
		102,
		"paid",
	)
	if err != nil {
		t.Fatalf("enqueue helper returned error: %v", err)
	}
	if !skipped {
		t.Fatalf("expected task skipped for empty receiver email")
	}
}

func TestEnqueueOrderStatusEmailTaskIfEligibleEnqueueNormalReceiver(t *testing.T) {
	queueClient, err := queue.NewClient(nil)
	if err != nil {
		t.Fatalf("new queue client failed: %v", err)
	}
	t.Cleanup(func() {
		_ = queueClient.Close()
	})

	skipped, err := enqueueOrderStatusEmailTaskIfEligible(
		orderStatusEmailOrderRepoStub{receiver: "buyer@example.com"},
		queueClient,
		nil,
		config.EmailConfig{},
		103,
		"paid",
	)
	if err != nil {
		t.Fatalf("enqueue helper returned error: %v", err)
	}
	if skipped {
		t.Fatalf("expected task enqueued for normal receiver email")
	}
}

func TestEnqueueOrderStatusEmailTaskIfEligibleFallbackWhenLookupFailed(t *testing.T) {
	queueClient, err := queue.NewClient(nil)
	if err != nil {
		t.Fatalf("new queue client failed: %v", err)
	}
	t.Cleanup(func() {
		_ = queueClient.Close()
	})

	skipped, err := enqueueOrderStatusEmailTaskIfEligible(
		orderStatusEmailOrderRepoStub{err: errors.New("lookup failed")},
		queueClient,
		nil,
		config.EmailConfig{},
		104,
		"paid",
	)
	if err != nil {
		t.Fatalf("enqueue helper returned error: %v", err)
	}
	if skipped {
		t.Fatalf("expected fallback enqueue when receiver lookup failed")
	}
}

func TestEnqueueOrderStatusEmailTaskIfEligibleSkipWhenSMTPDisabled(t *testing.T) {
	queueClient, err := queue.NewClient(nil)
	if err != nil {
		t.Fatalf("new queue client failed: %v", err)
	}
	t.Cleanup(func() {
		_ = queueClient.Close()
	})

	repo := newMockSettingRepo()
	repo.store[constants.SettingKeySMTPConfig] = models.JSON{
		"enabled": false,
	}

	skipped, err := enqueueOrderStatusEmailTaskIfEligible(
		orderStatusEmailOrderRepoStub{receiver: "buyer@example.com"},
		queueClient,
		NewSettingService(repo),
		config.EmailConfig{Enabled: true},
		105,
		"paid",
	)
	if err != nil {
		t.Fatalf("enqueue helper returned error: %v", err)
	}
	if !skipped {
		t.Fatalf("expected task skipped when smtp disabled")
	}
}
