package exchange

import "github.com/shopspring/decimal"

// Instrument contains contract specification for a trading symbol on an exchange.
type Instrument struct {
	Symbol       string
	VolStep      decimal.Decimal
	MinVol       decimal.Decimal
	PriceStep    decimal.Decimal
	ContractSize decimal.Decimal // 1 for ByBit/BingX (qty in coins), >1 for MEXC (qty in contracts)
}
