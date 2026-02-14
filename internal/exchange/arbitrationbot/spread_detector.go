package arbitrationbot

import (
	"fmt"
	"math"
	"time"

	"github.com/lucrumx/bot/internal/config"
)

// SpreadDetector detects arbitrage opportunities by comparing prices across exchanges for a given symbol.
type SpreadDetector struct {
	minSpreadPercent       float64              // Minimal spread percent
	maxAgeMs               int64                // Max age of price in milliseconds
	cooldownSignalDuration time.Duration        // Cooldown for signals
	cooldownSignal         map[string]time.Time // Cooldown for signals (not send before)
}

// PricePoint represents a price point with timestamp.
type PricePoint struct {
	Price float64
	TsMs  int64
}

// SpreadSignal represents an arbitrage opportunity.
type SpreadSignal struct {
	Symbol         string
	BuyOnExchange  string
	SellOnExchange string
	BuyPrice       float64
	SellPrice      float64
	SpreadPercent  float64
}

// NewSpreadDetector creates a new SpreadDetector.
func NewSpreadDetector(cfg *config.Config) *SpreadDetector {
	return &SpreadDetector{
		minSpreadPercent:       cfg.Exchange.ArbitrationBot.MinSpreadPercent,
		maxAgeMs:               cfg.Exchange.ArbitrationBot.MaxAgeMs,
		cooldownSignalDuration: cfg.Exchange.ArbitrationBot.CooldownSignal,
		cooldownSignal:         make(map[string]time.Time),
	}
}

// Detect identifies arbitrage opportunities by comparing prices across exchanges for a given symbol.
// pricesByExchange map[string]PricePoint - prices by exchanges for one symbol
// example pricesByExchange map[string]PricePoint:
//
//	{
//		"ByBit": {
//		   Price: 100, TsMs: 100
//		 },
//		"BingX": {
//		   Price: 100, TsMs: 100
//		 },
//	}
func (d *SpreadDetector) Detect(symbol string, pricesByExchange map[string]PricePoint) (*SpreadSignal, bool) {
	now := time.Now()

	freshestPrice := make(map[string]PricePoint, len(pricesByExchange))

	for exchangeName, price := range pricesByExchange {
		if price.Price <= 0 {
			continue
		}
		if now.UnixMilli()-price.TsMs > d.maxAgeMs {
			continue
		}

		freshestPrice[exchangeName] = price
	}

	// Need at least 2 exchanges to calculate spread
	if len(freshestPrice) < 2 {
		return nil, false
	}

	var spread *SpreadSignal
	var cooldownKey string
	var bestCooldownKey string

	for buyExchange, buyPrice := range freshestPrice {
		for sellExchange, sellPrice := range freshestPrice {

			if buyExchange == sellExchange {
				continue
			}

			cooldownKey = fmt.Sprintf("%s_%s_%s", symbol, buyExchange, sellExchange)
			if cooldown, ok := d.cooldownSignal[cooldownKey]; ok {
				if cooldown.Add(d.cooldownSignalDuration).After(now) {
					continue
				}
			}

			// TODO сейчас порог это gross. Добавить расчет net порога с учтом sell fee, buy fee,
			// TODO какое-нибудь проскальзываение ...
			spreadPercent := (sellPrice.Price - buyPrice.Price) / buyPrice.Price * 100

			if spreadPercent < d.minSpreadPercent {
				continue
			}

			if spread == nil || spreadPercent > spread.SpreadPercent {
				bestCooldownKey = cooldownKey
				spread = &SpreadSignal{
					Symbol:         symbol,
					BuyOnExchange:  buyExchange,
					SellOnExchange: sellExchange,
					BuyPrice:       buyPrice.Price,
					SellPrice:      sellPrice.Price,
					SpreadPercent:  spreadPercent,
				}
			}
		}
	}

	if spread == nil || math.IsNaN(spread.SpreadPercent) || math.IsInf(spread.SpreadPercent, 0) {
		return nil, false
	}

	d.cooldownSignal[bestCooldownKey] = time.Now()

	return spread, true
}
