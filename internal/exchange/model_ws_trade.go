package exchange

import "github.com/shopspring/decimal"

// Side represents the direction of a trade, such as "Buy" or "Sell".
type Side string

const (

	// Buy represents the "Buy" direction of a trade within the Side type.
	Buy Side = "Buy"
	// Sell represents the "Sell" direction of a trade within the Side type.
	Sell Side = "Sell"
)

// Trade represents a websocket trade with details like symbol, timestamp, price, volume, side, and USDT amount.
type Trade struct {
	Symbol     string
	Ts         int64
	Price      decimal.Decimal
	Volume     decimal.Decimal
	Side       Side
	USDTAmount decimal.Decimal // price * volume
}
