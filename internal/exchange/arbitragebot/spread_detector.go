package arbitragebot

import (
	"time"

	"github.com/lucrumx/bot/internal/config"
	"github.com/lucrumx/bot/internal/models"
)

// PricePoint represents a price point with timestamp.
type PricePoint struct {
	Price float64
	TsMs  int64
}

const minStepChangeToUpdate = 0.5

type ActiveSpreadState struct {
	maxSpreadPercent float64
}

type SpreadEvent struct {
	Status models.ArbitrageSpreadStatus

	Symbol         string
	BuyOnExchange  string
	SellOnExchange string

	BuyPrice  float64
	SellPrice float64

	FromSpreadPercent float64
	MaxSpreadPercent  float64
}

// SpreadDetector detects arbitrage opportunities by comparing prices across exchanges for a given symbol.
// NOT safe for concurrent use. Caller must ensure single-goroutine access.
type SpreadDetector struct {
	minSpreadPercent      float64                       // Minimal spread percent
	maxAgeMs              int64                         // Max age of price in milliseconds
	nowFn                 func() time.Time              // Function to get current time (for testing)
	activeSpreads         map[string]*ActiveSpreadState // current active spreads
	percentForCloseSpread float64                       // Spread percent for close signal
}

// NewSpreadDetector creates a new SpreadDetector.
func NewSpreadDetector(cfg *config.Config) *SpreadDetector {
	return &SpreadDetector{
		minSpreadPercent:      cfg.Exchange.ArbitrageBot.MinSpreadPercent,
		maxAgeMs:              cfg.Exchange.ArbitrageBot.MaxAgeMs,
		nowFn:                 time.Now,
		activeSpreads:         make(map[string]*ActiveSpreadState),
		percentForCloseSpread: cfg.Exchange.ArbitrageBot.PercentForCloseSpread,
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
func (d *SpreadDetector) Detect(symbol string, pricesByExchange map[string]PricePoint) []*SpreadEvent {
	now := d.nowFn()

	freshestPrice := d.filterFreshestPrices(pricesByExchange, now)
	if freshestPrice == nil {
		return nil
	}

	var spreadEvents []*SpreadEvent
	var spreadKey string

	for buyExchange, buyPrice := range freshestPrice {
		for sellExchange, sellPrice := range freshestPrice {

			if buyExchange == sellExchange {
				continue
			}

			if sellPrice.Price <= buyPrice.Price {
				// here spread will be negative
				continue
			}

			spreadKey = getSpreadKey(symbol, buyExchange, sellExchange)

			// TODO сейчас порог это gross. Добавить расчет net порога с учетом sell fee, buy fee,
			// TODO какое-нибудь проскальзываение ...
			spreadPercent := (sellPrice.Price - buyPrice.Price) / buyPrice.Price * 100

			//
			_, ok := d.activeSpreads[spreadKey]
			if !ok && spreadPercent < d.minSpreadPercent {
				continue
			} else if !ok && spreadPercent >= d.minSpreadPercent {
				// New spread
				d.activeSpreads[spreadKey] = &ActiveSpreadState{
					maxSpreadPercent: spreadPercent,
				}
				spreadEvents = append(spreadEvents, &SpreadEvent{
					Status:            models.ArbitrageSpreadOpened,
					Symbol:            symbol,
					BuyOnExchange:     buyExchange,
					SellOnExchange:    sellExchange,
					BuyPrice:          buyPrice.Price,
					SellPrice:         sellPrice.Price,
					FromSpreadPercent: spreadPercent,
					MaxSpreadPercent:  spreadPercent,
				})
			} else if ok && spreadPercent > d.activeSpreads[spreadKey].maxSpreadPercent+minStepChangeToUpdate {
				// Update
				d.activeSpreads[spreadKey].maxSpreadPercent = spreadPercent
				spreadEvents = append(spreadEvents, &SpreadEvent{
					Status:           models.ArbitrageSpreadUpdated,
					Symbol:           symbol,
					BuyOnExchange:    buyExchange,
					SellOnExchange:   sellExchange,
					MaxSpreadPercent: spreadPercent,
				})
			} else if ok && spreadPercent <= d.percentForCloseSpread {
				// Close
				spreadEvents = append(spreadEvents, &SpreadEvent{
					Status:         models.ArbitrageSpreadClosed,
					Symbol:         symbol,
					BuyOnExchange:  buyExchange,
					SellOnExchange: sellExchange,
				})

				delete(d.activeSpreads, spreadKey)
			}
		}
	}

	return spreadEvents
}

func (d *SpreadDetector) filterFreshestPrices(pricesByExchange map[string]PricePoint, now time.Time) map[string]PricePoint {
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
		return nil
	}

	return freshestPrice
}

func getSpreadKey(symbol string, buyOnExchange string, sellOnExchange string) string {
	return symbol + "#" + buyOnExchange + "#" + sellOnExchange
}
