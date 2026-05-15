package arbitragebot

import (
	"fmt"
	"time"

	"github.com/shopspring/decimal"

	"github.com/lucrumx/bot/internal/models"
)

// LimitStrategy opens positions with limit orders at last price and closes by market.
// If open legs don't fill within FillTimeoutDuration, they are cancelled and
// any filled leg is emergency-closed at market price.
type LimitStrategy struct {
	FillTimeoutDuration time.Duration
}

// OpenPrice returns the current last price as the limit price for open legs.
// Buy leg uses buy price, sell leg uses sell price from the spread event.
func (s LimitStrategy) OpenPrice(event *SpreadEvent, side models.OrderSide) *decimal.Decimal {
	var price float64
	if side == models.OrderSideBuy {
		price = event.BuyPrice
	} else {
		price = event.SellPrice
	}
	d := decimal.NewFromFloat(price)
	return &d
}

// ClosePrice returns nil — close legs are always executed at market price
// to guarantee position neutralization regardless of spread state.
func (s LimitStrategy) ClosePrice(_ *Position, _ models.OrderSide) *decimal.Decimal {
	return nil
}

// FillTimeout returns how long to wait for open legs to fill before cancelling.
func (s LimitStrategy) FillTimeout() time.Duration {
	return s.FillTimeoutDuration
}

// Validate checks that FillTimeoutDuration is set to a positive value.
func (s LimitStrategy) Validate() error {
	if s.FillTimeoutDuration <= 0 {
		return fmt.Errorf("LimitStrategy: FillTimeoutDuration must be > 0, got %s", s.FillTimeoutDuration)
	}
	return nil
}
