package arbitragebot

import (
	"time"

	"github.com/shopspring/decimal"

	"github.com/lucrumx/bot/internal/models"
)

// MarketStrategy executes both open and close legs as market orders with no fill timeout.
// Returns nil prices everywhere so Engine.buildOrder constructs market-type orders.
type MarketStrategy struct{}

// OpenPrice returns nil — open legs are submitted as market orders.
func (MarketStrategy) OpenPrice(_ *SpreadEvent, _ models.OrderSide) *decimal.Decimal {
	return nil
}

// ClosePrice returns nil — close legs are submitted as market orders.
func (MarketStrategy) ClosePrice(_ *Position, _ models.OrderSide) *decimal.Decimal {
	return nil
}

// FillTimeout returns 0 because market orders fill instantly; no watcher is needed.
func (MarketStrategy) FillTimeout() time.Duration {
	return 0
}

// Validate is a no-op for MarketStrategy — there is nothing to misconfigure.
func (MarketStrategy) Validate() error {
	return nil
}
