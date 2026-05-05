package arbitragebot

import (
	"time"

	"github.com/shopspring/decimal"

	"github.com/lucrumx/bot/internal/models"
)

// OrderStrategy determines how orders are priced and how long to wait for fills.
type OrderStrategy interface {
	// OpenPrice returns the limit price for an open leg, or nil for market order.
	OpenPrice(event *SpreadEvent, side models.OrderSide) *decimal.Decimal
	// ClosePrice returns the limit price for a close leg, or nil for market order.
	ClosePrice(pos *Position, side models.OrderSide) *decimal.Decimal
	// FillTimeout is how long to wait for open legs to fill before cancelling.
	// 0 means no timeout (market orders fill immediately).
	FillTimeout() time.Duration
	// Validate checks that the strategy is correctly configured.
	// Called at bot startup; returns an error if configuration is invalid.
	Validate() error
}
