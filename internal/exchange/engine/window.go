package engine

import (
	"sync"
	"time"

	"github.com/shopspring/decimal"

	"github.com/lucrumx/bot/internal/exchange"
)

const windowSize = 300 // пять минут

type tradeBucket struct {
	timestamp         int64           // Unix timestamp in seconds
	volumeUSDT        decimal.Decimal // Объем в
	count             int64
	openPrice         decimal.Decimal
	closePrice        decimal.Decimal
	OpenInterestValue decimal.Decimal
}

// Window represents a time window of trades.
type Window struct {
	mu      sync.Mutex
	buckets [windowSize]tradeBucket
}

// NewWindow creates a new time window.
func NewWindow() *Window {
	return &Window{
		buckets: [windowSize]tradeBucket{},
		mu:      sync.Mutex{},
	}
}

// AddTrade adds a new trade to the appropriate bucket in the time window, updating trade statistics for that bucket.
func (w *Window) AddTrade(trade exchange.Trade) {
	w.mu.Lock()
	defer w.mu.Unlock()

	//
	tsSec := trade.Ts / 1000
	idx := tsSec % windowSize // остаток от деления на размер окна, всегда будет от 0 до 59, потому что кратно минуте

	bucket := &w.buckets[idx]

	if bucket.timestamp == tsSec { // bucket устарел, тк минута та же (idx), но секунда другая, поэтмоу нужно обнулить
		bucket.timestamp = tsSec
		bucket.volumeUSDT = decimal.Zero
		bucket.count = 0
		bucket.openPrice = trade.Price
		bucket.closePrice = trade.Price
		bucket.OpenInterestValue = decimal.Zero
	}

	bucket.volumeUSDT = bucket.volumeUSDT.Add(trade.USDTAmount)
	bucket.count++
	bucket.closePrice = trade.Price
}

// Statistics represents aggregate trade statistics over a given number of seconds from the time window.
type Statistics struct {
	totalVolumeUSDT decimal.Decimal
	tradeCount      int64
	priceChangePcnt decimal.Decimal
}

// GetStatistics calculates aggregate trade statistics over the given number of seconds from the time window.
func (w *Window) GetStatistics(seconds int) (d Statistics) {
	w.mu.Lock()
	defer w.mu.Unlock()

	now := time.Now().Unix()
	var totalVolumeUSDT decimal.Decimal
	var tradeCount int64
	var openPrice decimal.Decimal
	var closePrice decimal.Decimal
	atLeastOneTradeFound := false

	for i := 0; i < seconds; i++ {
		ts := now - int64(i) // если i = 0, то now - 0 - это текущая секунда
		idx := ts % windowSize
		bucket := &w.buckets[idx]

		if bucket.timestamp != ts { // если bucket устарел или пуст, пропускаем
			continue
		}

		totalVolumeUSDT = totalVolumeUSDT.Add(bucket.volumeUSDT)
		tradeCount += bucket.count

		if i == 0 {
			closePrice = bucket.closePrice
		}

		openPrice = bucket.openPrice

		atLeastOneTradeFound = true
	}

	if !atLeastOneTradeFound || openPrice.IsZero() {
		return Statistics{
			totalVolumeUSDT: decimal.Zero,
			tradeCount:      0,
			priceChangePcnt: decimal.Zero,
		}
	}

	priceDelta := closePrice.Sub(openPrice).Div(openPrice)

	return Statistics{
		totalVolumeUSDT: totalVolumeUSDT,
		tradeCount:      tradeCount,
		priceChangePcnt: priceDelta.Mul(decimal.NewFromFloat(100)),
	}
}
