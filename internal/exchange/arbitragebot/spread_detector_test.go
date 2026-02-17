package arbitragebot

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/lucrumx/bot/internal/config"
	"github.com/lucrumx/bot/internal/models"
)

func getConfig() *config.Config {
	return &config.Config{
		Exchange: config.ExchangeConfig{
			ArbitrageBot: config.ArbitrageBotConfig{
				MaxAgeMs:              60_000,
				MinSpreadPercent:      1,
				PercentForCloseSpread: 0.1,
			},
		},
	}
}

func TestSpreadDetector_TestOpenSpreadEvent(t *testing.T) {
	cfg := getConfig()

	sd := NewSpreadDetector(cfg)

	spreadEvent := sd.Detect("BTCUSDT", map[string]PricePoint{
		"ByBit": {Price: 100, TsMs: time.Now().UnixMilli()},
		"BingX": {Price: 103, TsMs: time.Now().UnixMilli()},
	})

	expectedSpreadEvent := []*SpreadEvent{
		{
			Status:            models.ArbitrageSpreadOpened,
			Symbol:            "BTCUSDT",
			BuyOnExchange:     "ByBit",
			SellOnExchange:    "BingX",
			BuyPrice:          100,
			SellPrice:         103,
			FromSpreadPercent: 3.0,
			MaxSpreadPercent:  3.0,
		},
	}

	assert.Equal(t, expectedSpreadEvent, spreadEvent)
}

func TestSpreadDetector_TestUpdateSpreadEvent(t *testing.T) {
	cfg := getConfig()

	sd := NewSpreadDetector(cfg)

	_ = sd.Detect("BTCUSDT", map[string]PricePoint{
		"ByBit": {Price: 100, TsMs: time.Now().UnixMilli()},
		"BingX": {Price: 103, TsMs: time.Now().UnixMilli()},
	})

	updatedEvent := sd.Detect("BTCUSDT", map[string]PricePoint{
		"ByBit": {Price: 100, TsMs: time.Now().UnixMilli()},
		"BingX": {Price: 105, TsMs: time.Now().UnixMilli()},
	})
	expectedSpreadEvent := []*SpreadEvent{
		{
			Status:           models.ArbitrageSpreadUpdated,
			Symbol:           "BTCUSDT",
			BuyOnExchange:    "ByBit",
			SellOnExchange:   "BingX",
			BuyPrice:         100,
			SellPrice:        105,
			MaxSpreadPercent: 5.0,
		},
	}

	assert.Equal(t, expectedSpreadEvent, updatedEvent)
}

func TestSpreadDetector_TestCloseSpreadEvent(t *testing.T) {
	cfg := getConfig()

	sd := NewSpreadDetector(cfg)

	_ = sd.Detect("BTCUSDT", map[string]PricePoint{
		"ByBit": {Price: 100, TsMs: time.Now().UnixMilli()},
		"BingX": {Price: 103, TsMs: time.Now().UnixMilli()},
	})

	updatedEvent := sd.Detect("BTCUSDT", map[string]PricePoint{
		"ByBit": {Price: 100, TsMs: time.Now().UnixMilli()},
		"BingX": {Price: 100.001, TsMs: time.Now().UnixMilli()},
	})
	expectedSpreadEvent := []*SpreadEvent{
		{
			Status:         models.ArbitrageSpreadClosed,
			Symbol:         "BTCUSDT",
			BuyOnExchange:  "ByBit",
			SellOnExchange: "BingX",
		},
	}

	assert.Equal(t, expectedSpreadEvent, updatedEvent)
}
