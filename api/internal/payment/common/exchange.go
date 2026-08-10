package common

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"
)

// ExchangeRateConfig ??????,?????????? Config ??
type ExchangeRateConfig struct {
	TargetCurrency string `json:"target_currency"`
	ExchangeRate   string `json:"exchange_rate"`
}

// NormalizeExchangeRate ??????????
func (c *ExchangeRateConfig) NormalizeExchangeRate() {
	c.TargetCurrency = strings.ToUpper(strings.TrimSpace(c.TargetCurrency))
	c.ExchangeRate = strings.TrimSpace(c.ExchangeRate)
}

// NeedsCurrencyConversion ?????????
func (c *ExchangeRateConfig) NeedsCurrencyConversion() bool {
	return c.TargetCurrency != "" && c.ExchangeRate != ""
}

// ConvertAmount ??????????????????
// precision ??????(????? 2)?
// ?????????????????
func (c *ExchangeRateConfig) ConvertAmount(amount, currency string, precision int32) (string, string, error) {
	if !c.NeedsCurrencyConversion() {
		return amount, currency, nil
	}
	amountDec, err := decimal.NewFromString(strings.TrimSpace(amount))
	if err != nil {
		return "", "", fmt.Errorf("invalid amount %q", amount)
	}
	rate, err := decimal.NewFromString(c.ExchangeRate)
	if err != nil || rate.LessThanOrEqual(decimal.Zero) {
		return "", "", fmt.Errorf("invalid exchange_rate %q", c.ExchangeRate)
	}
	converted := amountDec.Mul(rate).Round(precision)
	return converted.String(), c.TargetCurrency, nil
}
