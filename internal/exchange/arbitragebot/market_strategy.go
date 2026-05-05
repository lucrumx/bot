package arbitragebot

import (
	"time"

	"github.com/shopspring/decimal"

	"github.com/lucrumx/bot/internal/models"
)

// MarketStrategy executes orders at market price with no fill timeout.
type MarketStrategy struct{}

func (MarketStrategy) OpenPrice(_ *SpreadEvent, _ models.OrderSide) *decimal.Decimal {
	return nil
}

func (MarketStrategy) ClosePrice(_ *Position, _ models.OrderSide) *decimal.Decimal {
	return nil
}

func (MarketStrategy) FillTimeout() time.Duration {
	return 0
}

func (MarketStrategy) Validate() error {
	return nil
}
