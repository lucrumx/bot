package utils

import (
	"github.com/shopspring/decimal"
)

// Decimal is a wrapper around decimal.Decimal that allows unmarshaling from JSON strings.
type Decimal struct {
	decimal.Decimal
}

// UnmarshalJSON implements the json.Unmarshaler interface. Should be used if float64 comes as a string with double quotes.
func (d *Decimal) UnmarshalJSON(data []byte) error {
	s := string(data)
	if s == `""` || s == "null" {
		d.Decimal = decimal.Zero
		return nil
	}
	return d.Decimal.UnmarshalJSON(data)
}

// MarshalJSON implements the json.Marshaler interface. Marshals Decimal as a string to preserve precision.
func (d Decimal) MarshalJSON() ([]byte, error) {
	return d.Decimal.MarshalJSON()
}
