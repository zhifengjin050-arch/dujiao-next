package models

import (
	"database/sql/driver"
	"encoding/json"

	"github.com/shopspring/decimal"
)

// Money ??????(?? 2 ???)
type Money struct {
	decimal.Decimal
}

// NewMoneyFromDecimal ? decimal ????
func NewMoneyFromDecimal(amount decimal.Decimal) Money {
	return Money{Decimal: amount.Round(2)}
}

// MarshalJSON ???? 2 ???????
func (m Money) MarshalJSON() ([]byte, error) {
	return json.Marshal(m.Decimal.Round(2).StringFixed(2))
}

// UnmarshalJSON ????(??????)
func (m *Money) UnmarshalJSON(b []byte) error {
	if len(b) == 0 {
		return nil
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		d, err := decimal.NewFromString(s)
		if err != nil {
			return err
		}
		m.Decimal = d.Round(2)
		return nil
	}
	var f float64
	if err := json.Unmarshal(b, &f); err != nil {
		return err
	}
	m.Decimal = decimal.NewFromFloat(f).Round(2)
	return nil
}

// Value ???????
func (m Money) Value() (driver.Value, error) {
	return m.Decimal.Round(2).Value()
}

// Scan ???????
func (m *Money) Scan(value interface{}) error {
	if err := m.Decimal.Scan(value); err != nil {
		return err
	}
	m.Decimal = m.Decimal.Round(2)
	return nil
}

// String ?? 2 ?????
func (m Money) String() string {
	return m.Decimal.Round(2).StringFixed(2)
}
