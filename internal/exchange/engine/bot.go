// Package engine contains the bot engine.
package engine

import (
	"context"
	"fmt"
	"hash/fnv"
	"math"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"

	"github.com/lucrumx/bot/internal/notifier"

	"github.com/lucrumx/bot/internal/utils"

	"github.com/lucrumx/bot/internal/exchange"
)

// Bot represents a bot engine.
type Bot struct {
	provider exchange.Provider

	workers []*worker

	filterTickersByTurnover decimal.Decimal
	pumpInterval            int
	targetPriceChange       float64
	startupDelay            time.Duration
	checkInterval           time.Duration
	alertStep               decimal.Decimal

	startTime time.Time

	logger   zerolog.Logger
	notifier notifier.Notifier

	rpsTimerIntervalInSec int
	tradeCounter          uint64
}

// NewBot creates a new Bot (constructor).
func NewBot(provider exchange.Provider, notif notifier.Notifier) *Bot {
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

	checkIntervalRaw, err := strconv.Atoi(utils.GetEnv("CHECK_INTERVAL", ""))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse CHECK_INTERVAL evn")
	}

	alertStep, err := decimal.NewFromString(utils.GetEnv("ALERT_STEP", ""))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse ALERT_STEP evn")
	}

	rpsTimerIntervalInSec, err := strconv.Atoi(utils.GetEnv("RPS_TIMER_INTERVAL", "60"))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse RPS_TIMER_INTERVAL evn")
	}

	return &Bot{
		provider: provider,

		filterTickersByTurnover: filterTickersByTurnover,
		pumpInterval:            pumpInterval,
		targetPriceChange:       targetPriceChange,
		startupDelay:            time.Duration(startupDelay) * time.Second,
		checkInterval:           time.Duration(checkIntervalRaw) * time.Second,
		alertStep:               alertStep,

		rpsTimerIntervalInSec: rpsTimerIntervalInSec,

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
	cntTickers := len(*tickers)
	if cntTickers == 0 {
		return nil, fmt.Errorf("bot engine: no tickers found")
	}
	b.logger.Info().Msgf("bot engine: got %d tickers", cntTickers)

	filteredTickers := b.filterTickers(*tickers)

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

		if ticker.Turnover24h.GreaterThan(b.filterTickersByTurnover) {
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
