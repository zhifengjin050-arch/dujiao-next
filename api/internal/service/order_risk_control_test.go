package service

import (
	"testing"

	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/repository"

	"gorm.io/gorm"
)

// --- NormalizeOrderRiskControlConfig ?? ---

func TestNormalizeOrderRiskControlConfig_Defaults(t *testing.T) {
	cfg := NormalizeOrderRiskControlConfig(OrderRiskControlConfig{})
	if cfg.MaxPendingOrdersPerUser != 0 {
		t.Fatalf("zero should be preserved (means no limit), got %d", cfg.MaxPendingOrdersPerUser)
	}
}

func TestNormalizeOrderRiskControlConfig_ClampValues(t *testing.T) {
	cfg := NormalizeOrderRiskControlConfig(OrderRiskControlConfig{
		MaxPendingOrdersPerUser:       -1,
		MaxPendingOrdersPerIP:         200,
		MaxPendingOrdersPerGuestEmail: 50,
		OrderRateLimit: OrderRateLimitConfig{
			WindowSeconds: 5,    // below min 10
			MaxRequests:   0,    // below min 1
			BlockSeconds:  -100, // below min 0
		},
	})
	if cfg.MaxPendingOrdersPerUser != 3 {
		t.Fatalf("expected clamped to default 3, got %d", cfg.MaxPendingOrdersPerUser)
	}
	if cfg.MaxPendingOrdersPerIP != 5 {
		t.Fatalf("expected clamped to default 5, got %d", cfg.MaxPendingOrdersPerIP)
	}
	if cfg.MaxPendingOrdersPerGuestEmail != 50 {
		t.Fatalf("expected 50 (valid), got %d", cfg.MaxPendingOrdersPerGuestEmail)
	}
	if cfg.OrderRateLimit.WindowSeconds != 60 {
		t.Fatalf("expected clamped window to 60, got %d", cfg.OrderRateLimit.WindowSeconds)
	}
	if cfg.OrderRateLimit.MaxRequests != 5 {
		t.Fatalf("expected clamped max_requests to 5, got %d", cfg.OrderRateLimit.MaxRequests)
	}
	if cfg.OrderRateLimit.BlockSeconds != 120 {
		t.Fatalf("expected clamped block to 120, got %d", cfg.OrderRateLimit.BlockSeconds)
	}
}

func TestNormalizeOrderRiskControlConfig_IPValidation(t *testing.T) {
	cfg := NormalizeOrderRiskControlConfig(OrderRiskControlConfig{
		IPBlacklist: []string{
			"1.2.3.4",         // valid IP
			"10.0.0.0/8",      // valid CIDR
			"invalid_ip",      // invalid - should be removed
			"",                // empty - should be removed
			"  192.168.1.1  ", // valid with whitespace
			"999.999.999.999", // invalid IP
			"abc/24",          // invalid CIDR
		},
	})
	expected := []string{"1.2.3.4", "10.0.0.0/8", "192.168.1.1"}
	if len(cfg.IPBlacklist) != len(expected) {
		t.Fatalf("expected %d IPs, got %d: %v", len(expected), len(cfg.IPBlacklist), cfg.IPBlacklist)
	}
	for i, ip := range expected {
		if cfg.IPBlacklist[i] != ip {
			t.Fatalf("expected IP[%d]=%q, got %q", i, ip, cfg.IPBlacklist[i])
		}
	}
}

func TestNormalizeOrderRiskControlConfig_EmailNormalization(t *testing.T) {
	cfg := NormalizeOrderRiskControlConfig(OrderRiskControlConfig{
		EmailBlacklist: []string{
			"  Spam@Example.COM  ",
			"",
			"test@test.com",
		},
	})
	if len(cfg.EmailBlacklist) != 2 {
		t.Fatalf("expected 2 emails, got %d: %v", len(cfg.EmailBlacklist), cfg.EmailBlacklist)
	}
	if cfg.EmailBlacklist[0] != "spam@example.com" {
		t.Fatalf("expected lowercased email, got %q", cfg.EmailBlacklist[0])
	}
}

// --- isValidIPOrCIDR ?? ---

func TestIsValidIPOrCIDR(t *testing.T) {
	tests := []struct {
		input string
		valid bool
	}{
		{"1.2.3.4", true},
		{"::1", true},
		{"10.0.0.0/8", true},
		{"192.168.0.0/16", true},
		{"fe80::/10", true},
		{"invalid", false},
		{"999.999.999.999", false},
		{"abc/24", false},
		{"", false},
	}
	for _, tc := range tests {
		if got := isValidIPOrCIDR(tc.input); got != tc.valid {
			t.Errorf("isValidIPOrCIDR(%q) = %v, want %v", tc.input, got, tc.valid)
		}
	}
}

// --- isIPInBlacklist ?? ---

func TestIsIPInBlacklist(t *testing.T) {
	svc := &OrderRiskControlService{}

	blacklist := []string{"1.2.3.4", "10.0.0.0/8", "192.168.1.0/24"}

	tests := []struct {
		ip      string
		blocked bool
	}{
		{"1.2.3.4", true},        // exact match
		{"1.2.3.5", false},       // not in list
		{"10.0.0.1", true},       // CIDR /8 match
		{"10.255.255.255", true}, // CIDR /8 match
		{"11.0.0.1", false},      // outside CIDR
		{"192.168.1.100", true},  // CIDR /24 match
		{"192.168.2.1", false},   // outside CIDR /24
		{"", false},              // empty IP
		{"invalid", false},       // invalid IP
	}

	for _, tc := range tests {
		if got := svc.isIPInBlacklist(tc.ip, blacklist); got != tc.blocked {
			t.Errorf("isIPInBlacklist(%q) = %v, want %v", tc.ip, got, tc.blocked)
		}
	}
}

func TestIsIPInBlacklist_CacheReuse(t *testing.T) {
	svc := &OrderRiskControlService{}
	blacklist := []string{"1.2.3.4", "10.0.0.0/8"}

	// First call builds cache
	svc.isIPInBlacklist("1.2.3.4", blacklist)
	if svc.cachedBlacklist == nil {
		t.Fatal("expected cache to be built")
	}
	hash1 := svc.cachedBlacklist.hash

	// Same list should reuse cache
	svc.isIPInBlacklist("10.0.0.1", blacklist)
	if svc.cachedBlacklist.hash != hash1 {
		t.Fatal("expected cache to be reused")
	}

	// Different list should rebuild cache
	svc.isIPInBlacklist("1.2.3.4", []string{"5.6.7.8"})
	if svc.cachedBlacklist.hash == hash1 {
		t.Fatal("expected cache to be rebuilt for different list")
	}
}

// --- RiskRateLimitedError ?? ---

func TestRiskRateLimitedError_Is(t *testing.T) {
	err := &RiskRateLimitedError{RetryAfter: 60}
	if err.Error() != ErrRiskOrderRateLimited.Error() {
		t.Fatalf("expected error message match")
	}
	if !err.Is(ErrRiskOrderRateLimited) {
		t.Fatal("expected Is(ErrRiskOrderRateLimited) to be true")
	}
}

func TestGetRetryAfter(t *testing.T) {
	if ra := GetRetryAfter(ErrRiskIPBlacklisted); ra != 0 {
		t.Fatalf("expected 0 for non-rate-limit error, got %d", ra)
	}
	if ra := GetRetryAfter(&RiskRateLimitedError{RetryAfter: 42}); ra != 42 {
		t.Fatalf("expected 42, got %d", ra)
	}
}

// --- CheckOrderAllowed ????(?? mock) ---

type mockOrderRepoForRisk struct {
	repository.OrderRepository
	pendingByUser  int64
	pendingByIP    int64
	pendingByEmail int64
}

func (m *mockOrderRepoForRisk) CountPendingByUserID(_ uint) (int64, error) {
	return m.pendingByUser, nil
}
func (m *mockOrderRepoForRisk) CountPendingByClientIP(_ string) (int64, error) {
	return m.pendingByIP, nil
}
func (m *mockOrderRepoForRisk) CountPendingByGuestEmail(_ string) (int64, error) {
	return m.pendingByEmail, nil
}

// Implement remaining interface methods as no-ops
func (m *mockOrderRepoForRisk) Create(_ *models.Order, _ []models.OrderItem) error { return nil }
func (m *mockOrderRepoForRisk) GetByID(_ uint) (*models.Order, error)              { return nil, nil }
func (m *mockOrderRepoForRisk) GetByIDs(_ []uint) ([]models.Order, error)          { return nil, nil }
func (m *mockOrderRepoForRisk) ResolveReceiverEmailByOrderID(_ uint) (string, error) {
	return "", nil
}
func (m *mockOrderRepoForRisk) GetByIDAndUser(_ uint, _ uint) (*models.Order, error) {
	return nil, nil
}
func (m *mockOrderRepoForRisk) GetByOrderNoAndUser(_ string, _ uint) (*models.Order, error) {
	return nil, nil
}
func (m *mockOrderRepoForRisk) GetAnyByOrderNoAndUser(_ string, _ uint) (*models.Order, error) {
	return nil, nil
}
func (m *mockOrderRepoForRisk) GetByIDAndGuest(_ uint, _, _ string) (*models.Order, error) {
	return nil, nil
}
func (m *mockOrderRepoForRisk) GetByOrderNoAndGuest(_, _, _ string) (*models.Order, error) {
	return nil, nil
}
func (m *mockOrderRepoForRisk) GetAnyByOrderNoAndGuest(_, _, _ string) (*models.Order, error) {
	return nil, nil
}
func (m *mockOrderRepoForRisk) ListChildren(_ uint) ([]models.Order, error) { return nil, nil }
func (m *mockOrderRepoForRisk) ListByUser(_ repository.OrderListFilter) ([]models.Order, int64, error) {
	return nil, 0, nil
}
func (m *mockOrderRepoForRisk) ListByGuest(_, _ string, _, _ int) ([]models.Order, int64, error) {
	return nil, 0, nil
}
func (m *mockOrderRepoForRisk) ListAdmin(_ repository.OrderListFilter) ([]models.Order, int64, error) {
	return nil, 0, nil
}
func (m *mockOrderRepoForRisk) UpdateStatus(_ uint, _ string, _ map[string]interface{}) error {
	return nil
}
func (m *mockOrderRepoForRisk) CountOrderItemsByProduct(_ uint) (int64, error) { return 0, nil }
func (m *mockOrderRepoForRisk) Transaction(_ func(tx *gorm.DB) error) error    { return nil }
func (m *mockOrderRepoForRisk) WithTx(_ *gorm.DB) *repository.GormOrderRepository {
	return nil
}

func newTestRiskControlService(pendingByUser, pendingByIP, pendingByEmail int64, cfgJSON models.JSON) *OrderRiskControlService {
	settingRepo := newMockSettingRepo()
	if cfgJSON != nil {
		settingRepo.Upsert("order_risk_control_config", cfgJSON)
	}
	settingSvc := NewSettingService(settingRepo)
	return NewOrderRiskControlService(settingSvc, &mockOrderRepoForRisk{
		pendingByUser:  pendingByUser,
		pendingByIP:    pendingByIP,
		pendingByEmail: pendingByEmail,
	})
}

func TestCheckOrderAllowed_DisabledByDefault(t *testing.T) {
	svc := newTestRiskControlService(100, 100, 100, nil)
	err := svc.CheckOrderAllowed(RiskCheckInput{
		UserID:   1,
		ClientIP: "1.2.3.4",
	})
	if err != nil {
		t.Fatalf("expected nil when disabled, got %v", err)
	}
}

func TestCheckOrderAllowed_IPBlacklist(t *testing.T) {
	svc := newTestRiskControlService(0, 0, 0, models.JSON{
		"enabled":      true,
		"ip_blacklist": []interface{}{"1.2.3.4", "10.0.0.0/8"},
	})

	if err := svc.CheckOrderAllowed(RiskCheckInput{ClientIP: "1.2.3.4"}); err != ErrRiskIPBlacklisted {
		t.Fatalf("expected ErrRiskIPBlacklisted, got %v", err)
	}
	if err := svc.CheckOrderAllowed(RiskCheckInput{ClientIP: "10.0.0.1"}); err != ErrRiskIPBlacklisted {
		t.Fatalf("expected ErrRiskIPBlacklisted for CIDR, got %v", err)
	}
	if err := svc.CheckOrderAllowed(RiskCheckInput{ClientIP: "2.3.4.5"}); err != nil {
		t.Fatalf("expected nil for non-blocked IP, got %v", err)
	}
}

func TestCheckOrderAllowed_EmailBlacklist(t *testing.T) {
	svc := newTestRiskControlService(0, 0, 0, models.JSON{
		"enabled":         true,
		"email_blacklist": []interface{}{"spam@example.com"},
	})

	if err := svc.CheckOrderAllowed(RiskCheckInput{
		IsGuest:    true,
		GuestEmail: "SPAM@example.com",
		ClientIP:   "2.3.4.5",
	}); err != ErrRiskEmailBlacklisted {
		t.Fatalf("expected ErrRiskEmailBlacklisted, got %v", err)
	}

	// Non-guest should not be blocked by email blacklist
	if err := svc.CheckOrderAllowed(RiskCheckInput{
		UserID:   1,
		ClientIP: "2.3.4.5",
	}); err != nil {
		t.Fatalf("expected nil for non-guest, got %v", err)
	}
}

func TestCheckOrderAllowed_PendingOrderLimits(t *testing.T) {
	cfg := models.JSON{
		"enabled":                            true,
		"max_pending_orders_per_user":        float64(2),
		"max_pending_orders_per_ip":          float64(3),
		"max_pending_orders_per_guest_email": float64(1),
	}

	// User at limit
	svc := newTestRiskControlService(2, 0, 0, cfg)
	if err := svc.CheckOrderAllowed(RiskCheckInput{UserID: 1, ClientIP: "5.6.7.8"}); err != ErrRiskTooManyPendingOrders {
		t.Fatalf("expected ErrRiskTooManyPendingOrders for user, got %v", err)
	}

	// User under limit
	svc = newTestRiskControlService(1, 0, 0, cfg)
	if err := svc.CheckOrderAllowed(RiskCheckInput{UserID: 1, ClientIP: "5.6.7.8"}); err != nil {
		t.Fatalf("expected nil for user under limit, got %v", err)
	}

	// IP at limit
	svc = newTestRiskControlService(0, 3, 0, cfg)
	if err := svc.CheckOrderAllowed(RiskCheckInput{ClientIP: "5.6.7.8"}); err != ErrRiskTooManyPendingOrders {
		t.Fatalf("expected ErrRiskTooManyPendingOrders for IP, got %v", err)
	}

	// Guest email at limit
	svc = newTestRiskControlService(0, 0, 1, cfg)
	if err := svc.CheckOrderAllowed(RiskCheckInput{
		IsGuest:    true,
		GuestEmail: "test@example.com",
		ClientIP:   "5.6.7.8",
	}); err != ErrRiskTooManyPendingOrders {
		t.Fatalf("expected ErrRiskTooManyPendingOrders for guest email, got %v", err)
	}
}

func TestCheckOrderAllowed_SkipIPCheck(t *testing.T) {
	cfg := models.JSON{
		"enabled":                     true,
		"max_pending_orders_per_user": float64(5),
		"max_pending_orders_per_ip":   float64(1),
		"ip_blacklist":                []interface{}{"1.2.3.4"},
	}

	// IP ?????,? SkipIPCheck=true ???
	svc := newTestRiskControlService(0, 0, 0, cfg)
	if err := svc.CheckOrderAllowed(RiskCheckInput{
		UserID:      1,
		ClientIP:    "1.2.3.4",
		SkipIPCheck: true,
	}); err != nil {
		t.Fatalf("expected nil with SkipIPCheck, got %v", err)
	}

	// IP ????,? SkipIPCheck=true ???
	svc = newTestRiskControlService(0, 999, 0, cfg)
	if err := svc.CheckOrderAllowed(RiskCheckInput{
		UserID:      1,
		ClientIP:    "5.6.7.8",
		SkipIPCheck: true,
	}); err != nil {
		t.Fatalf("expected nil with SkipIPCheck for IP pending, got %v", err)
	}

	// ??????,SkipIPCheck=true ???,????
	svc = newTestRiskControlService(5, 0, 0, cfg)
	if err := svc.CheckOrderAllowed(RiskCheckInput{
		UserID:      1,
		ClientIP:    "5.6.7.8",
		SkipIPCheck: true,
	}); err != ErrRiskTooManyPendingOrders {
		t.Fatalf("expected ErrRiskTooManyPendingOrders for user limit with SkipIPCheck, got %v", err)
	}
}

func TestCheckOrderAllowed_ZeroLimitMeansNoLimit(t *testing.T) {
	svc := newTestRiskControlService(999, 999, 999, models.JSON{
		"enabled":                            true,
		"max_pending_orders_per_user":        float64(0),
		"max_pending_orders_per_ip":          float64(0),
		"max_pending_orders_per_guest_email": float64(0),
	})
	if err := svc.CheckOrderAllowed(RiskCheckInput{
		UserID:     1,
		ClientIP:   "1.2.3.4",
		IsGuest:    true,
		GuestEmail: "test@example.com",
	}); err != nil {
		t.Fatalf("expected nil when limits are 0 (disabled), got %v", err)
	}
}
