// Package manipulationbot implements a bot that detects manipulation - pump spot (atr on spot more then perpetual).
package manipulationbot

import (
	"context"
	"fmt"
	"hash/fnv"
	"runtime"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"

	"github.com/lucrumx/bot/internal/exchange"
	"github.com/lucrumx/bot/internal/notifier"
)

// Bot consumes spot and perp trades of one exchange and emits manipulation alerts.
type Bot struct {
	provider exchange.Provider
	notifier notifier.Notifier
	logger   zerolog.Logger
	cfg      Config
	detector *detector
	workers  []*worker
	started  time.Time

	tradeCounter uint64
}

// NewBot constructs the manipulation detector bot.
func NewBot(provider exchange.Provider, notif notifier.Notifier, cfg Config, logger zerolog.Logger) *Bot {
	return &Bot{
		provider: provider,
		notifier: notif,
		logger:   logger,
		cfg:      cfg,
		detector: newDetector(cfg),
	}
}

// Run starts subscriptions and processing loop.
func (b *Bot) Run(ctx context.Context) error {
	b.started = time.Now()

	symbols, err := b.resolveSymbols(ctx)
	if err != nil {
		return err
	}
	if len(symbols) == 0 {
		return fmt.Errorf("manipulation bot: no symbols to monitor")
	}

	b.logger.Info().
		Int("symbols", len(symbols)).
		Str("exchange", b.provider.GetExchangeName()).
		Msg("manipulation bot: symbols selected")

	spotTrades, err := b.provider.SubscribeTrades(ctx, symbols, exchange.CategorySpot)
	if err != nil {
		return fmt.Errorf("manipulation bot: subscribe spot trades: %w", err)
	}

	perpTrades, err := b.provider.SubscribeTrades(ctx, symbols, exchange.CategoryLinear)
	if err != nil {
		return fmt.Errorf("manipulation bot: subscribe linear trades: %w", err)
	}

	numWorkers := runtime.NumCPU()
	b.workers = make([]*worker, numWorkers)
	workerCtx, cancelWorkers := context.WithCancel(ctx)
	defer cancelWorkers()

	for i := 0; i < numWorkers; i++ {
		b.workers[i] = &worker{
			bot:    b,
			inChan: make(chan exchange.Trade, 50_000),
			states: make(map[string]*symbolState),
		}
		go b.workers[i].run(workerCtx)
	}

	go b.logTradeRate(ctx)

	errCh := make(chan error, 2)
	go b.forwardTrades(workerCtx, spotTrades, errCh)
	go b.forwardTrades(workerCtx, perpTrades, errCh)

	select {
	case <-ctx.Done():
		return nil
	case err := <-errCh:
		cancelWorkers()
		return err
	}
}

func (b *Bot) resolveSymbols(ctx context.Context) ([]string, error) {
	if len(b.cfg.Symbols) > 0 {
		b.logger.Info().
			Int("symbols", len(b.cfg.Symbols)).
			Msg("manipulation bot: using symbols from config")
		return b.cfg.Symbols, nil
	}

	spotTickers, err := b.provider.GetTickers(ctx, nil, exchange.CategorySpot)
	if err != nil {
		return nil, fmt.Errorf("manipulation bot: get spot tickers: %w", err)
	}

	perpTickers, err := b.provider.GetTickers(ctx, nil, exchange.CategoryLinear)
	if err != nil {
		return nil, fmt.Errorf("manipulation bot: get linear tickers: %w", err)
	}

	spotBySymbol := make(map[string]exchange.Ticker, len(spotTickers))
	spotSymbols := make(map[string]struct{}, len(spotTickers))
	spotUSDTCount := 0
	for _, ticker := range spotTickers {
		if !strings.HasSuffix(ticker.Symbol, "USDT") {
			continue
		}
		spotUSDTCount++
		spotBySymbol[ticker.Symbol] = ticker
		spotSymbols[ticker.Symbol] = struct{}{}
	}

	symbols := make([]string, 0)
	perpSymbols := make(map[string]struct{}, len(perpTickers))
	perpUSDTCount := 0
	missingSpotCount := 0
	filteredByPerpTurnover := 0
	filteredBySpotTurnover := 0
	missingSpotSymbols := make([]string, 0)
	for _, ticker := range perpTickers {
		if !strings.HasSuffix(ticker.Symbol, "USDT") {
			continue
		}
		perpUSDTCount++
		perpSymbols[ticker.Symbol] = struct{}{}

		spotTicker, ok := spotBySymbol[ticker.Symbol]
		if !ok {
			missingSpotCount++
			missingSpotSymbols = append(missingSpotSymbols, ticker.Symbol)
			continue
		}

		if b.cfg.MinPerpTurnover24h > 0 && ticker.Turnover24h.InexactFloat64() < b.cfg.MinPerpTurnover24h {
			filteredByPerpTurnover++
			continue
		}

		if b.cfg.MaxSpotTurnover24h > 0 && spotTicker.Turnover24h.InexactFloat64() > b.cfg.MaxSpotTurnover24h {
			filteredBySpotTurnover++
			continue
		}

		symbols = append(symbols, ticker.Symbol)
	}

	missingPerpSymbols := make([]string, 0)
	for symbol := range spotSymbols {
		if _, ok := perpSymbols[symbol]; ok {
			continue
		}
		missingPerpSymbols = append(missingPerpSymbols, symbol)
	}

	sort.Strings(missingSpotSymbols)
	sort.Strings(missingPerpSymbols)

	if len(missingSpotSymbols) > 0 {
		b.logger.Info().
			Int("count", len(missingSpotSymbols)).
			Str("symbols", strings.Join(missingSpotSymbols, ",")).
			Msg("manipulation bot: perp symbols without spot market")
	}

	if len(missingPerpSymbols) > 0 {
		b.logger.Info().
			Int("count", len(missingPerpSymbols)).
			Str("symbols", strings.Join(missingPerpSymbols, ",")).
			Msg("manipulation bot: spot symbols without perp market")
	}

	b.logger.Info().
		Int("spot_usdt", spotUSDTCount).
		Int("perp_usdt", perpUSDTCount).
		Int("missing_spot", missingSpotCount).
		Int("filtered_by_min_perp_turnover", filteredByPerpTurnover).
		Int("filtered_by_max_spot_turnover", filteredBySpotTurnover).
		Int("selected", len(symbols)).
		Float64("min_perp_turnover_24h", b.cfg.MinPerpTurnover24h).
		Float64("max_spot_turnover_24h", b.cfg.MaxSpotTurnover24h).
		Msg("manipulation bot: symbol selection stats")

	return symbols, nil
}

func (b *Bot) forwardTrades(ctx context.Context, source <-chan exchange.Trade, errCh chan<- error) {
	hasher := fnv.New32a()

	for {
		select {
		case <-ctx.Done():
			return
		case trade, ok := <-source:
			if !ok {
				select {
				case errCh <- fmt.Errorf("manipulation bot: trade channel closed"):
				default:
				}
				return
			}

			atomic.AddUint64(&b.tradeCounter, 1)
			hasher.Reset()
			_, _ = hasher.Write([]byte(trade.Symbol))

			workerIdx := hasher.Sum32() % uint32(len(b.workers))
			select {
			case b.workers[workerIdx].inChan <- trade:
			default:
				b.logger.Warn().
					Str("symbol", trade.Symbol).
					Str("category", string(trade.Category)).
					Msg("manipulation bot: worker queue full, drop trade")
			}
		}
	}
}

func (b *Bot) logTradeRate(ctx context.Context) {
	ticker := time.NewTicker(b.cfg.RPSTimerInterval)
	defer ticker.Stop()

	var lastCount uint64

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			current := atomic.LoadUint64(&b.tradeCounter)
			diff := current - lastCount
			lastCount = current

			rps := float64(diff) / b.cfg.RPSTimerInterval.Seconds()
			if diff == 0 {
				continue
			}

			b.logger.Info().
				Uint64("total", current).
				Uint64("delta", diff).
				Float64("rps", rps).
				Msg("manipulation bot throughput")
		}
	}
}

type worker struct {
	bot    *Bot
	inChan chan exchange.Trade
	states map[string]*symbolState
}

func (w *worker) run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case trade, ok := <-w.inChan:
			if !ok {
				return
			}
			w.processTrade(trade)
		}
	}
}

func (w *worker) processTrade(trade exchange.Trade) {
	state, ok := w.states[trade.Symbol]
	if !ok {
		state = newSymbolState(w.bot.cfg)
		w.states[trade.Symbol] = state
	}

	switch trade.Category {
	case exchange.CategorySpot:
		state.spot.AddTrade(trade)
	case exchange.CategoryLinear:
		state.perp.AddTrade(trade)
	default:
		return
	}

	sig := w.bot.detector.evaluate(trade.Symbol, state, w.bot.started)
	if sig == nil {
		return
	}

	w.bot.logger.Warn().
		Str("symbol", sig.Symbol).
		Float64("spot_atr", sig.SpotATR).
		Float64("perp_atr", sig.PerpATR).
		Float64("spot_atr_pct", sig.SpotATRPct).
		Float64("perp_atr_pct", sig.PerpATRPct).
		Float64("atr_ratio", sig.ATRRatio).
		Msg("atr spot-vs-perp signal")

	if err := w.bot.notifier.Send(sig.Message(w.bot.provider.GetExchangeName())); err != nil {
		w.bot.logger.Warn().Err(err).Msg("manipulation bot: send notification failed")
	}
}
