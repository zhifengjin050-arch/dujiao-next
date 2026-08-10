package common

import (
	"testing"
)

func TestNormalizeExchangeRate(t *testing.T) {
	c := ExchangeRateConfig{
		TargetCurrency: "  cny  ",
		ExchangeRate:   " 7.2 ",
	}
	c.NormalizeExchangeRate()
	if c.TargetCurrency != "CNY" {
		t.Errorf("TargetCurrency = %q, want CNY", c.TargetCurrency)
	}
	if c.ExchangeRate != "7.2" {
		t.Errorf("ExchangeRate = %q, want 7.2", c.ExchangeRate)
	}
}

func TestNeedsCurrencyConversion(t *testing.T) {
	tests := []struct {
		name     string
		target   string
		rate     string
		expected bool
	}{
		{"both set", "CNY", "7.2", true},
		{"empty target", "", "7.2", false},
		{"empty rate", "CNY", "", false},
		{"both empty", "", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := ExchangeRateConfig{TargetCurrency: tt.target, ExchangeRate: tt.rate}
			if got := c.NeedsCurrencyConversion(); got != tt.expected {
				t.Errorf("NeedsCurrencyConversion() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestConvertAmount(t *testing.T) {
	tests := []struct {
		name           string
		target         string
		rate           string
		amount         string
		currency       string
		precision      int32
		wantAmount     string
		wantCurrency   string
		wantErr        bool
		skipConversion bool // no conversion expected
	}{
		{
			name:   "USD to CNY precision 2",
			target: "CNY", rate: "7.2",
			amount: "10", currency: "USD", precision: 2,
			wantAmount: "72", wantCurrency: "CNY",
		},
		{
			name:   "fractional amount",
			target: "CNY", rate: "7.25",
			amount: "9.99", currency: "USD", precision: 2,
			wantAmount: "72.43", wantCurrency: "CNY",
		},
		{
			name:   "high precision for crypto",
			target: "USDT", rate: "0.13888",
			amount: "100", currency: "CNY", precision: 8,
			wantAmount: "13.888", wantCurrency: "USDT",
		},
		{
			name:   "no conversion when not configured",
			target: "", rate: "",
			amount: "10", currency: "USD", precision: 2,
			wantAmount: "10", wantCurrency: "USD",
			skipConversion: true,
		},
		{
			name:   "invalid amount",
			target: "CNY", rate: "7.2",
			amount: "abc", currency: "USD", precision: 2,
			wantErr: true,
		},
		{
			name:   "invalid rate",
			target: "CNY", rate: "xyz",
			amount: "10", currency: "USD", precision: 2,
			wantErr: true,
		},
		{
			name:   "zero rate",
			target: "CNY", rate: "0",
			amount: "10", currency: "USD", precision: 2,
			wantErr: true,
		},
		{
			name:   "negative rate",
			target: "CNY", rate: "-1.5",
			amount: "10", currency: "USD", precision: 2,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := ExchangeRateConfig{TargetCurrency: tt.target, ExchangeRate: tt.rate}
			gotAmount, gotCurrency, err := c.ConvertAmount(tt.amount, tt.currency, tt.precision)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotAmount != tt.wantAmount {
				t.Errorf("amount = %q, want %q", gotAmount, tt.wantAmount)
			}
			if gotCurrency != tt.wantCurrency {
				t.Errorf("currency = %q, want %q", gotCurrency, tt.wantCurrency)
			}
		})
	}
}
