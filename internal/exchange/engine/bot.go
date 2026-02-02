// Package engine contains the bot engine.
package engine

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"

	"github.com/lucrumx/bot/internal/utils"

	"github.com/lucrumx/bot/internal/exchange"
)

// Bot represents a bot engine.
type Bot struct {
	provider exchange.Provider

	mutex   sync.Mutex
	windows map[string]*Window

	filterTickersByTurnover decimal.Decimal
	pumpInterval            int
	targetPriceChange       float64
	startupDelay            time.Duration

	startTime time.Time

	logger zerolog.Logger
}

// NewBot creates a new Bot (constructor).
func NewBot(provider exchange.Provider) *Bot {
	rawTurnover := strings.ReplaceAll(utils.GetEnv("FILTER_TICKERS_TURNOVER", ""), "_", "")
	filterTickersByTurnover, err := decimal.NewFromString(rawTurnover)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse FILTER_TICKERS_TURNOVER evn")
	}

	pumpInterval, err := strconv.Atoi(utils.GetEnv("PUMP_INTERVAL", ""))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse PUMP_INTERVAL evn")
	}

	targetPriceChange, err := strconv.ParseFloat(utils.GetEnv("TARGET_PRICE_CHANGE", ""), 64)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse TARGET_PRICE_CHANGE evn")
	}

	startupDelay, err := strconv.ParseFloat(utils.GetEnv("STARTUP_DELAY", ""), 64)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse STARTUP_DELAY evn")
	}

	return &Bot{
		provider: provider,
		mutex:    sync.Mutex{},
		windows:  map[string]*Window{},

		filterTickersByTurnover: filterTickersByTurnover,
		pumpInterval:            pumpInterval,
		targetPriceChange:       targetPriceChange,
		startupDelay:            time.Duration(startupDelay) * time.Second,

		logger: log.Output(zerolog.ConsoleWriter{Out: os.Stderr}),
	}
}

// StartBot starts the bot engine and returns a channel of trades.
func (b *Bot) StartBot(ctx context.Context) (<-chan exchange.Trade, error) {
	b.startTime = time.Now()
	b.logger.Info().Msg("bot engine: starting bot")

	log.Print("bot engine: getting tickers")
	tickers, err := b.provider.GetTickers(ctx, nil, exchange.CategoryLinear)
	if err != nil {
		return nil, fmt.Errorf("bot engine: failed to get tickers")
	}
	cntTickers := len(*tickers)
	if cntTickers == 0 {
		return nil, fmt.Errorf("bot engine: no tickers found")
	}

	filteredTickers := b.filterTickers(*tickers)

	b.logger.Info().Msg("bot engine: filtered tickers")
	b.logger.Info().Msg(strings.Join(filteredTickers, ","))

	sourceChan, err := b.provider.SubscribeTrades(ctx, filteredTickers)
	if err != nil {
		return nil, err
	}

	b.logger.Info().Msgf("bot engine: starting trade processor and collection statistics for %d seconds", b.pumpInterval)

	outChan := make(chan exchange.Trade, 10000)

	go func() {
		defer close(outChan)
		for {
			select {
			case <-ctx.Done():
				return
			case trade, ok := <-sourceChan:
				if !ok {
					return
				}
				b.processTrade(trade)

				// Пробрасываем дальше
				select {
				case outChan <- trade:
				default:
					// если обработка outChan тормозит- данные пропадут
				}
			}
		}
	}()

	return outChan, nil
}

func (b *Bot) filterTickers(tickers []exchange.Ticker) []string {
	b.logger.Info().Msgf("bot engine: got %d tickers", len(tickers))

	filteredTickers := make([]string, 0, len(tickers))
	for _, ticker := range tickers {
		if !strings.HasSuffix(ticker.Symbol, "USDT") {
			continue
		}

		if ticker.Turnover24h.GreaterThan(b.filterTickersByTurnover) {
			continue
		}

		filteredTickers = append(filteredTickers, ticker.Symbol)
	}

	b.logger.Info().Msgf("bot engine: %d tickers left after filtering", len(filteredTickers))
	return filteredTickers
}

func (b *Bot) processTrade(trade exchange.Trade) {
	b.mutex.Lock()
	window, ok := b.windows[trade.Symbol]
	if !ok {
		window = NewWindow(b.pumpInterval)
		b.windows[trade.Symbol] = window
	}
	b.mutex.Unlock()

	window.AddTrade(trade)
	b.checkPump(trade.Symbol, window)
}

func (b *Bot) checkPump(symbol string, win *Window) {
	if time.Since(b.startTime) < b.startupDelay {
		return
	}

	stats := win.GetStatistics(b.pumpInterval)

	threshPrice := decimal.NewFromFloat(b.targetPriceChange)

	if stats.priceChangePcnt.GreaterThanOrEqual(threshPrice) {
		b.logger.Warn().
			Str("pair", symbol).
			Str("period", "15m").
			Str("price_change", stats.priceChangePcnt.StringFixed(2)+"%").
			Str("volume_15m", stats.totalVolumeUSDT.StringFixed(0)).
			Int64("trades_15m", stats.tradeCount).
			Msg("🔥 STRONG PUMP DETECTED")
	}
}
