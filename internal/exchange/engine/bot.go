// Package engine contains the bot engine.
package engine

import (
	"context"
	"fmt"
	"hash/fnv"
	"math"
	"os"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"

	"github.com/lucrumx/bot/internal/config"

	"github.com/lucrumx/bot/internal/notifier"

	"github.com/lucrumx/bot/internal/exchange"
)

// Bot represents a bot engine.
type Bot struct {
	provider exchange.Provider

	workers []*worker

	filterTickersByTurnover float64
	pumpInterval            int
	targetPriceChange       float64
	startupDelay            time.Duration
	checkInterval           time.Duration
	alertStep               float64

	startTime time.Time

	logger   zerolog.Logger
	notifier notifier.Notifier

	rpsTimerIntervalInSec int
	tradeCounter          uint64
}

// NewBot creates a new Bot (constructor).
func NewBot(provider exchange.Provider, notif notifier.Notifier, cfg *config.Config) *Bot {
	return &Bot{
		provider: provider,

		filterTickersByTurnover: cfg.Exchange.Bot.FilterTickersTurnover,
		pumpInterval:            cfg.Exchange.Bot.PumpInterval,
		targetPriceChange:       cfg.Exchange.Bot.TargetPriceChange,
		startupDelay:            cfg.Exchange.Bot.StartupDelay,
		checkInterval:           cfg.Exchange.Bot.CheckInterval,
		alertStep:               cfg.Exchange.Bot.AlertStep,

		rpsTimerIntervalInSec: cfg.Exchange.Bot.RpsTimerInterval,

		logger:   log.Output(zerolog.ConsoleWriter{Out: os.Stderr}),
		notifier: notif,
	}
}

// StartBot starts the bot engine and returns a channel of trades.
func (b *Bot) StartBot(ctx context.Context) (<-chan exchange.Trade, error) {
	b.startTime = time.Now()
	b.logger.Info().Msg("bot engine: starting bot and getting tickers")

	tickers, err := b.provider.GetTickers(ctx, nil, exchange.CategoryLinear)
	if err != nil {
		return nil, fmt.Errorf("bot engine: failed to get tickers")
	}
	if tickers == nil {
		return nil, fmt.Errorf("tickers not found")
	}
	cntTickers := len(tickers)
	if cntTickers == 0 {
		return nil, fmt.Errorf("bot engine: no tickers found")
	}
	b.logger.Info().Msgf("bot engine: got %d tickers", cntTickers)

	filteredTickers := b.filterTickers(tickers)

	sourceChan, err := b.provider.SubscribeTrades(ctx, filteredTickers)
	if err != nil {
		return nil, err
	}

	b.logger.Info().Msgf("bot engine: starting trade processor and collection statistics for %d seconds", b.pumpInterval)

	outChan := make(chan exchange.Trade, 200_000)

	numWorkers := runtime.NumCPU()
	b.workers = make([]*worker, numWorkers)

	workerCtx, cancelWorkers := context.WithCancel(ctx)

	for i := 0; i < numWorkers; i++ {
		b.workers[i] = &worker{
			id:      i,
			bot:     b,
			inChan:  make(chan exchange.Trade, 50_000),
			windows: make(map[string]*Window),
		}
		go b.workers[i].workerStart(workerCtx)
	}

	b.logger.Info().Msgf("bot engine: started %d workers", numWorkers)

	go b.tradeCount(ctx)

	go func() {
		defer close(outChan)

		hasher := fnv.New32a()

		for {
			select {
			case <-ctx.Done():
				cancelWorkers()
				return
			case trade, ok := <-sourceChan:
				if !ok {
					cancelWorkers()
					return
				}

				hasher.Reset()
				_, _ = hasher.Write([]byte(trade.Symbol))
				hash := hasher.Sum32()

				workerIdx := hash % uint32(numWorkers)
				select {
				case b.workers[workerIdx].inChan <- trade:
					//
				default:
					b.logger.Warn().Str("s", trade.Symbol).Msg("worker dropped trade")
				}

				select {
				case outChan <- trade:
				default:
				}
			}
		}
	}()

	return outChan, nil
}

func (b *Bot) filterTickers(tickers []exchange.Ticker) []string {
	filteredTickers := make([]string, 0, len(tickers))
	for _, ticker := range tickers {
		if !strings.HasSuffix(ticker.Symbol, "USDT") {
			continue
		}

		if ticker.Turnover24h.GreaterThan(decimal.NewFromFloat(b.filterTickersByTurnover)) {
			continue
		}

		filteredTickers = append(filteredTickers, ticker.Symbol)
	}

	b.logger.Info().Msgf("bot engine: %d tickers left after filtering", len(filteredTickers))
	return filteredTickers
}

func (b *Bot) tradeCount(ctx context.Context) {
	ticker := time.NewTicker(time.Second * time.Duration(b.rpsTimerIntervalInSec))
	defer ticker.Stop()

	lastCount := uint64(0)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			current := atomic.LoadUint64(&b.tradeCounter)
			diff := current - lastCount
			rps := float64(diff) / float64(b.rpsTimerIntervalInSec)

			if diff > 0 {
				b.logger.Info().
					Uint64("total", current).
					Uint64("delta", diff).
					Float64("rps", math.Round(rps)).
					Msg("Engine throughput")
			}

			lastCount = current
		}
	}
}
