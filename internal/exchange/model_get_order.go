package exchange

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ExchangeOrder represents an order on the exchange, with fields relevant for tracking its execution and calculating PnL.
//
//nolint:revive
type ExchangeOrder struct {
	OrderID         uuid.UUID
	ExchangeOrderID string
	ExchangeName    string
	AvgPrice        decimal.Decimal
	Fees            decimal.Decimal
	Profit          decimal.Decimal
}
