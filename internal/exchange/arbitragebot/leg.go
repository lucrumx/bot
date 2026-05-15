package arbitragebot

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Leg represents one side of an arbitrage position on a single exchange.
type Leg struct {
	OrderID         uuid.UUID
	ExchangeOrderID string // populated after CreateOrder, needed for CancelOrder on exchanges that require it (e.g. MEXC)

	// Populated on fill — used for partial fill tracking (limit orders)
	FilledQty decimal.Decimal
	AvgPrice  decimal.Decimal
	Confirmed bool
}
