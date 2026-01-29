package engine

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"github.com/lucrumx/bot/internal/exchange"
)

func TestWindow(t *testing.T) {
	window := NewWindow()

	nowTs := time.Now().UnixMilli()

	window.AddTrade(exchange.Trade{
		Symbol:     "BTCUSD",
		Price:      decimal.NewFromInt(100),
		Volume:     decimal.NewFromInt(100),
		USDTAmount: decimal.NewFromInt(100),
		Side:       exchange.Buy,
		Ts:         nowTs,
	})

	window.AddTrade(exchange.Trade{
		Symbol:     "BTCUSD",
		Price:      decimal.NewFromInt(101),
		Volume:     decimal.NewFromInt(100),
		USDTAmount: decimal.NewFromInt(101),
		Side:       exchange.Buy,
		Ts:         nowTs,
	})

	idx := (nowTs / 1000) % windowSize
	bucket := window.buckets[idx]

	assert.Equal(t, int64(2), bucket.count)
	assert.True(t, decimal.NewFromInt(201).Equal(bucket.volumeUSDT))
}
