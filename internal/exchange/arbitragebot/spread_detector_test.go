package arbitragebot

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/lucrumx/bot/internal/config"
)

const cooldown = 30 * time.Minute

func getConfig() *config.Config {
	return &config.Config{
		Exchange: config.ExchangeConfig{
			ArbitrageBot: config.ArbitrageBotConfig{
				MaxAgeMs:         60_000,
				MinSpreadPercent: 1,
				CooldownSignal:   cooldown,
			},
		},
	}
}

func TestSpreadDetector_TestSignal(t *testing.T) {
	cfg := getConfig()

	sd := NewSpreadDetector(cfg)

	signal, hasSignal := sd.Detect("BTCUSDT", map[string]PricePoint{
		"ByBit": {Price: 100, TsMs: time.Now().UnixMilli()},
		"BingX": {Price: 103, TsMs: time.Now().UnixMilli()},
	})

	expectedSignal := &SpreadSignal{
		Symbol:         "BTCUSDT",
		BuyOnExchange:  "ByBit",
		SellOnExchange: "BingX",
		BuyPrice:       100,
		SellPrice:      103,
		SpreadPercent:  3.0,
	}

	assert.Equal(t, true, hasSignal)
	assert.Equal(t, expectedSignal, signal)
}

func TestSpreadDetector_TestOldTradeCase(t *testing.T) {
	const symbol = "BTCUSDT"
	now := time.Now()
	cfg := getConfig()

	testCases := []struct {
		name         string
		prices       map[string]PricePoint
		expectSignal bool
		now          time.Time
	}{
		{
			name: "Test case 1: no cooldown",
			prices: map[string]PricePoint{
				"ByBit": {Price: 100, TsMs: time.Now().UnixMilli()},
				"BingX": {Price: 101, TsMs: time.Now().UnixMilli()},
			},
			now:          now,
			expectSignal: true,
		},
		{
			name: "Test case 2: old trade",
			prices: map[string]PricePoint{
				"ByBit": {Price: 100, TsMs: now.Add(-time.Duration(cfg.Exchange.ArbitrageBot.MaxAgeMs+1) * time.Millisecond).UnixMilli()},
				"BingX": {Price: 101, TsMs: now.Add(-time.Duration(cfg.Exchange.ArbitrageBot.MaxAgeMs+1) * time.Millisecond).UnixMilli()},
			},
			now:          now,
			expectSignal: false,
		},
	}

	sd := NewSpreadDetector(cfg)

	for _, tc := range testCases {
		sd.nowFn = func() time.Time {
			return tc.now
		}
		_, hasSignal := sd.Detect(symbol, tc.prices)

		assert.Equal(t, tc.expectSignal, hasSignal, tc.name)
	}
}

func TestSpreadDetector_TestCooldown(t *testing.T) {
	const symbol = "BTCUSDT"
	now := time.Now()

	testCases := []struct {
		name         string
		prices       map[string]PricePoint
		expectSignal bool
		now          time.Time
	}{
		{
			name: "Test case 1: no cooldown",
			prices: map[string]PricePoint{
				"ByBit": {Price: 100, TsMs: time.Now().UnixMilli()},
				"BingX": {Price: 101, TsMs: time.Now().UnixMilli()},
			},
			now:          now,
			expectSignal: true,
		},
		{
			name: "Test case 2: cooldown exists",
			prices: map[string]PricePoint{
				"ByBit": {Price: 100, TsMs: time.Now().UnixMilli()},
				"BingX": {Price: 101, TsMs: time.Now().UnixMilli()},
			},
			now:          now,
			expectSignal: false,
		},
		{
			name: "Test case 3: cooldown expired",
			prices: map[string]PricePoint{
				"ByBit": {Price: 100, TsMs: time.Now().Add(cooldown + time.Duration(1*time.Minute)).UnixMilli()},
				"BingX": {Price: 101, TsMs: time.Now().Add(cooldown + time.Duration(1*time.Minute)).UnixMilli()},
			},
			now:          now.Add(cooldown + 1*time.Minute),
			expectSignal: true,
		},
	}

	sd := NewSpreadDetector(getConfig())

	for _, tc := range testCases {
		sd.nowFn = func() time.Time {
			return tc.now
		}
		_, hasSignal := sd.Detect(symbol, tc.prices)

		assert.Equal(t, tc.expectSignal, hasSignal, tc.name)
	}
}
