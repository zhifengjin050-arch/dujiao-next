package epay

import (
	"testing"

	"github.com/dujiao-next/internal/constants"
)

func TestVerifyCallbackOwnershipV1Success(t *testing.T) {
	cfg := &Config{
		EpayVersion: VersionV1,
		MerchantID:  "1001",
		MerchantKey: "key-001",
	}
	form := signEpayV1CallbackForm(t, cfg, map[string]string{
		"pid":          "1001",
		"out_trade_no": "ORDER-1001",
		"trade_no":     "EPAY-2001",
		"money":        "10.00",
		"trade_status": constants.EpayTradeStatusSuccess,
	})

	if err := VerifyCallback(cfg, form); err != nil {
		t.Fatalf("verify callback failed: %v", err)
	}
	if err := VerifyCallbackOwnership(cfg, form); err != nil {
		t.Fatalf("verify callback ownership failed: %v", err)
	}
}

func TestVerifyCallbackOwnershipV1MissingPID(t *testing.T) {
	cfg := &Config{
		EpayVersion: VersionV1,
		MerchantID:  "1001",
	}
	form := map[string][]string{
		"out_trade_no": {"ORDER-1001"},
	}

	if err := VerifyCallbackOwnership(cfg, form); err == nil {
		t.Fatalf("expected ownership verify error for missing pid")
	}
}

func TestVerifyCallbackOwnershipV1PIDMismatch(t *testing.T) {
	cfg := &Config{
		EpayVersion: VersionV1,
		MerchantID:  "1001",
	}
	form := map[string][]string{
		"pid": {"1002"},
	}

	if err := VerifyCallbackOwnership(cfg, form); err == nil {
		t.Fatalf("expected ownership verify error for pid mismatch")
	}
}

func TestVerifyCallbackOwnershipV2Success(t *testing.T) {
	privateKeyPEM, publicKeyPEM := generateEpayRSAKeyPair(t)
	cfg := &Config{
		EpayVersion: VersionV2,
		MerchantID:  "2001",
		PrivateKey:  privateKeyPEM,
		PublicKey:   publicKeyPEM,
		SignType:    epaySignTypeRSA,
	}
	form := signEpayV2CallbackForm(t, cfg, map[string]string{
		"pid":          "2001",
		"out_trade_no": "ORDER-2001",
		"trade_no":     "EPAYV2-3001",
		"money":        "20.00",
		"trade_status": constants.EpayTradeStatusSuccess,
		"timestamp":    "1721206072",
	})

	if err := VerifyCallback(cfg, form); err != nil {
		t.Fatalf("verify callback failed: %v", err)
	}
	if err := VerifyCallbackOwnership(cfg, form); err != nil {
		t.Fatalf("verify callback ownership failed: %v", err)
	}
}

func TestVerifyCallbackOwnershipV2MissingPID(t *testing.T) {
	cfg := &Config{
		EpayVersion: VersionV2,
		MerchantID:  "2001",
	}
	form := map[string][]string{
		"out_trade_no": {"ORDER-2001"},
	}

	if err := VerifyCallbackOwnership(cfg, form); err == nil {
		t.Fatalf("expected ownership verify error for missing pid")
	}
}

func TestVerifyCallbackOwnershipV2PIDMismatch(t *testing.T) {
	cfg := &Config{
		EpayVersion: VersionV2,
		MerchantID:  "2001",
	}
	form := map[string][]string{
		"pid": {"2002"},
	}

	if err := VerifyCallbackOwnership(cfg, form); err == nil {
		t.Fatalf("expected ownership verify error for pid mismatch")
	}
}

func signEpayV1CallbackForm(t *testing.T, cfg *Config, params map[string]string) map[string][]string {
	t.Helper()

	form := make(map[string][]string, len(params)+2)
	for key, value := range params {
		form[key] = []string{value}
	}
	form["sign_type"] = []string{epaySignTypeMD5}
	form["sign"] = []string{signMD5(buildSignContent(params) + cfg.MerchantKey)}
	return form
}

func signEpayV2CallbackForm(t *testing.T, cfg *Config, params map[string]string) map[string][]string {
	t.Helper()

	sign, err := signRSA(buildSignContent(params), cfg.PrivateKey)
	if err != nil {
		t.Fatalf("sign rsa callback content failed: %v", err)
	}
	form := make(map[string][]string, len(params)+2)
	for key, value := range params {
		form[key] = []string{value}
	}
	form["sign_type"] = []string{epaySignTypeRSA}
	form["sign"] = []string{sign}
	return form
}
