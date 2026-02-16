// Package arbitragebot provides an arbitrage bot engine.
package arbitragebot

import (
	"context"
	"fmt"
	"strconv"

	"github.com/rs/zerolog"

	"github.com/lucrumx/bot/internal/config"

	"github.com/lucrumx/bot/internal/exchange"
	"github.com/lucrumx/bot/internal/notifier"
)

// ArbitrageBot represents a bot engine.
type ArbitrageBot struct {
	logger   zerolog.Logger
	notifier notifier.Notifier
	clients  []exchange.Provider
	// executor *ExecutionEngine order processing
	cfg *config.Config
}

// NewBot creates a new Bot (constructor).
func NewBot(client []exchange.Provider, notif notifier.Notifier, logger zerolog.Logger, cfg *config.Config) *ArbitrageBot {
	return &ArbitrageBot{
		logger:   logger,
		notifier: notif,
		clients:  client,
		cfg:      cfg,
	}
}

// PriceChangeEvent represents an event triggered by a change in price on a specific exchange.
type PriceChangeEvent struct {
	TsMs         int64
	ExchangeName string
	Symbol       string
	Price        float64
}

// Prices represent a map of prices for a specific symbol on different exchanges.
//
//	{
//	  "BTCUSDT": {
//	    "ByBit": { Price: 100, TsMs: 100 },
//	    "BingX": { Price: 101, TsMs: 101 },
//	  },
//	  "ETHUSDT": {
//	    "ByBit": { Price: 100, TsMs: 100 },
//	    "BingX": { Price: 101, TsMs: 101 },
//	  },
//	}
type Prices map[string]map[string]PricePoint

// Run starts the arbitrage bot engine.
func (a *ArbitrageBot) Run(ctx context.Context) error {
	if len(a.clients) < 2 {
		return fmt.Errorf("not enough clients: %d", len(a.clients))
	}

	uniq, notUniqName, uniqNames := checkUniqClient(a.clients)
	if !uniq {
		return fmt.Errorf("not uniq clients: %s", notUniqName)
	}
	a.logger.Info().Msgf("uniq clients: %s", uniqNames)

	symbolsByExchange := map[string]map[string]struct{}{}
	for _, client := range a.clients {
		exchangeName := client.GetExchangeName()

		tickers, err := client.GetTickers(ctx, []string{}, exchange.CategoryLinear)
		if err != nil {
			return fmt.Errorf("failed to get tickers from %s: %w", exchangeName, err)
		}

		symbols := map[string]struct{}{}
		for _, ticker := range tickers {
			if ticker.Symbol == "" {
				continue
			}
			symbols[ticker.Symbol] = struct{}{}
		}
		symbolsByExchange[exchangeName] = symbols
	}

	intersectedSymbols := intersect(symbolsByExchange) // common symbols for all exchanges
	if len(intersectedSymbols) < 1 {
		return fmt.Errorf("no common symbols for exchanges")
	}
	a.logger.Info().Msgf("common symbols: %d", len(intersectedSymbols))

	//
	// websocket subscription
	symbols := make([]string, 0, len(intersectedSymbols))
	for s := range intersectedSymbols {
		symbols = append(symbols, s)
	}

	tradeEventsCh := make(chan PriceChangeEvent, 2000)
	errCh := make(chan error, len(a.clients))
	go a.grabTrade(ctx, symbols, tradeEventsCh, errCh)

	prices := make(Prices)
	spreadDetector := NewSpreadDetector(a.cfg)

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-errCh:
			return fmt.Errorf("failed to grab trade: %w", err)
		case event := <-tradeEventsCh:
			if prices[event.Symbol] == nil {
				prices[event.Symbol] = map[string]PricePoint{}
			}
			prices[event.Symbol][event.ExchangeName] = PricePoint{
				Price: event.Price,
				TsMs:  event.TsMs,
			}
			spreadSignal, hasSpread := spreadDetector.Detect(event.Symbol, prices[event.Symbol])
			if hasSpread {
				a.handleSignal(spreadSignal)
			}
		}
	}
}

func (a *ArbitrageBot) grabTrade(ctx context.Context, symbols []string, tradeEventsCh chan<- PriceChangeEvent, errCh chan<- error) {
	subCtx, subCancel := context.WithCancel(ctx)
	defer subCancel()

	sendErrNonBlocking := func(err error) {
		select {
		case errCh <- err:
		default:
			a.logger.Warn().Err(err).Msg("drop err: errCh is full")
		}
	}

	for _, client := range a.clients {
		exchangeName := client.GetExchangeName()
		tradeCh, err := client.SubscribeTrades(subCtx, symbols)
		if err != nil {
			sendErrNonBlocking(fmt.Errorf("failed to subscribe to trades on %s: %w", exchangeName, err))
			return
		}

		a.logger.Info().Msgf("subscribed to trades on %s", exchangeName)

		go func(exchangeName string, ch <-chan exchange.Trade) {
			defer subCancel()
			for {
				select {
				case <-subCtx.Done():
					return
				case trade, ok := <-ch:
					if !ok {
						sendErrNonBlocking(fmt.Errorf("trade channel closed on %s", exchangeName))
						return
					}

					event := PriceChangeEvent{
						Symbol:       trade.Symbol,
						TsMs:         trade.Ts,
						ExchangeName: exchangeName,
						Price:        trade.Price,
					}

					select {
					case tradeEventsCh <- event:
					case <-subCtx.Done():
						return
					}
				}
			}
		}(exchangeName, tradeCh)
	}

	<-subCtx.Done()
}

func intersect(symbolsByExchange map[string]map[string]struct{}) map[string]struct{} {
	result := map[string]struct{}{}
	first := true

	for _, symbols := range symbolsByExchange {
		if first {
			for symbol := range symbols {
				result[symbol] = struct{}{}
			}
			first = false
			continue
		}

		for s := range result {
			if _, ok := symbols[s]; !ok {
				delete(result, s)
			}
		}
	}

	return result
}

func checkUniqClient(clients []exchange.Provider) (bool, string, string) {
	cnt := make(map[string]int)
	var names string
	for _, client := range clients {
		n := client.GetExchangeName()
		cnt[n]++

		if cnt[n] > 1 {
			return false, n, ""
		}

		if len(names) > 0 {
			names += ", "
		}
		names += n
	}

	return true, "", names
}

func (a *ArbitrageBot) handleSignal(spread *SpreadSignal) {
	spreadStr := strconv.FormatFloat(spread.SpreadPercent, 'f', 2, 64)

	a.logger.Warn().
		Str("pair", spread.Symbol).
		Str("spread", spreadStr).
		Str("buy on", spread.BuyOnExchange).
		Str("sell on", spread.SellOnExchange).
		Msg("🔥 SPREAD DETECTED")

	msg := fmt.Sprintf(
		"<b>🔔 ARBITRAGE: Ticker - %s</b>\n\n"+
			"Spread: <code>%s%%</code>\n\n"+
			"🟢 Buy:  %s - <b>%.4f</b>\n"+
			"🔴 Sell: %s - <b>%.4f</b>",
		spread.Symbol,
		spreadStr,
		spread.BuyOnExchange, spread.BuyPrice,
		spread.SellOnExchange, spread.SellPrice,
	)

	go func() {
		if err := a.notifier.Send(msg); err != nil {
			a.logger.Warn().Err(err).Msg("failed to send telegram notification")
		}
	}()

	// a.executor.Trade(ctx, spread)
}
