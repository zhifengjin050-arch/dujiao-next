package service

import (
	"testing"

	"github.com/dujiao-next/internal/constants"
)

func TestGetAffiliateSettingFallback(t *testing.T) {
	repo := newMockSettingRepo()
	svc := NewSettingService(repo)

	setting, err := svc.GetAffiliateSetting()
	if err != nil {
		t.Fatalf("get affiliate setting failed: %v", err)
	}
	if setting.Enabled {
		t.Fatalf("expected default enabled false")
	}
	if setting.CommissionRate != 0 {
		t.Fatalf("expected default commission rate 0, got %v", setting.CommissionRate)
	}
	if setting.ConfirmDays != 0 {
		t.Fatalf("expected default confirm days 0, got %d", setting.ConfirmDays)
	}
	if setting.MinWithdrawAmount != 0 {
		t.Fatalf("expected default min withdraw amount 0, got %v", setting.MinWithdrawAmount)
	}
	if len(setting.WithdrawChannels) != 0 {
		t.Fatalf("expected default withdraw channels empty, got %v", setting.WithdrawChannels)
	}
}

func TestUpdateAffiliateSettingNormalize(t *testing.T) {
	repo := newMockSettingRepo()
	svc := NewSettingService(repo)

	setting, err := svc.UpdateAffiliateSetting(AffiliateSetting{
		Enabled:           true,
		CommissionRate:    123.456,
		ConfirmDays:       -10,
		MinWithdrawAmount: -100.239,
		WithdrawChannels:  []string{"  usdt  ", "USDT", "", "paypal"},
	})
	if err != nil {
		t.Fatalf("update affiliate setting failed: %v", err)
	}
	if !setting.Enabled {
		t.Fatalf("expected enabled true")
	}
	if setting.CommissionRate != 100 {
		t.Fatalf("expected commission rate clamp to 100, got %v", setting.CommissionRate)
	}
	if setting.ConfirmDays != 0 {
		t.Fatalf("expected confirm days clamp to 0, got %d", setting.ConfirmDays)
	}
	if setting.MinWithdrawAmount != 0 {
		t.Fatalf("expected min withdraw amount clamp to 0, got %v", setting.MinWithdrawAmount)
	}
	if len(setting.WithdrawChannels) != 2 {
		t.Fatalf("expected 2 withdraw channels, got %v", setting.WithdrawChannels)
	}
	if setting.WithdrawChannels[0] != "usdt" || setting.WithdrawChannels[1] != "paypal" {
		t.Fatalf("unexpected withdraw channels: %v", setting.WithdrawChannels)
	}

	saved, ok := repo.store[constants.SettingKeyAffiliateConfig]
	if !ok {
		t.Fatalf("expected affiliate setting saved")
	}
	if saved["commission_rate"] != 100.0 {
		t.Fatalf("expected saved commission rate 100, got %v", saved["commission_rate"])
	}
}
