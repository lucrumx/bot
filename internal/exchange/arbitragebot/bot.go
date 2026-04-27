// Package arbitragebot provides an arbitrage bot engine.
package arbitragebot

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"

	"github.com/lucrumx/bot/internal/notifier"

	"github.com/lucrumx/bot/internal/config"

	"github.com/lucrumx/bot/internal/exchange"
)

const (
	minBalanceForTrading = 5
)

// ArbitrageBot represents a bot engine.
type ArbitrageBot struct {
	logger     zerolog.Logger
	clients    []exchange.Provider
	cfg        *config.Config
	tradeCount int64
	engine     *Engine
}

// NewBot creates a new Bot (constructor).
func NewBot(
	clients []exchange.Provider,
	logger zerolog.Logger,
	cfg *config.Config,
	notify notifier.Notifier,
	arbitrageSpreadRepo ArbitrageSpreadRepository,
	orderRepo OrderRepository,
) *ArbitrageBot {

	silentModeTxt := "off"
	if cfg.Exchange.ArbitrageBot.SilentMode {
		silentModeTxt = "on"
	}
	fmt.Printf("Arbitrage bot silent mode is %s\n", silentModeTxt)

	engine := NewEngine(cfg, clients, orderRepo, arbitrageSpreadRepo, notify, logger)
	return &ArbitrageBot{
		logger:  logger,
		clients: clients,
		cfg:     cfg,
		engine:  engine,
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
	a.clients = a.skipExchange()
	
	if len(a.clients) < 2 {
		return fmt.Errorf("not enough clients: %d", len(a.clients))
	}

	// Start retriving balances
	balanceStore := newBalanceStore(a.logger)
	balanceStore.Start(ctx, a.clients)
	a.checkBalances(ctx)

	if err := a.engine.LoadInstruments(ctx, a.clients); err != nil {
		return fmt.Errorf("failed to load instruments: %w", err)
	}
	a.logger.Info().Msg("instrument cache loaded")

	if err := a.engine.ListenExecutions(ctx); err != nil {
		return fmt.Errorf("failed to subscribe to executions: %w", err)
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

	go a.engine.Run(ctx)
	go a.grabTrade(ctx, symbols, tradeEventsCh, errCh)
	go a.logTradeCount(ctx)

	prices := make(Prices)
	spreadDetector := NewSpreadDetector(a.cfg)

	for {
		select {
		case <-ctx.Done():
			return nil
		case err := <-errCh:
			return fmt.Errorf("failed to grab trade: %w", err)
		case event := <-tradeEventsCh:
			atomic.AddInt64(&a.tradeCount, 1)

			if prices[event.Symbol] == nil {
				prices[event.Symbol] = map[string]PricePoint{}
			}
			prices[event.Symbol][event.ExchangeName] = PricePoint{
				Price: event.Price,
				TsMs:  event.TsMs,
			}
			spreadEvents := spreadDetector.Detect(event.Symbol, prices[event.Symbol])
			if spreadEvents != nil {
				a.engine.HandleSignal(spreadEvents)
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
		tradeCh, err := client.SubscribeTrades(subCtx, symbols, exchange.CategoryLinear)
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

func (a *ArbitrageBot) logTradeCount(ctx context.Context) {
	ticker := time.NewTicker(time.Second * time.Duration(a.cfg.Exchange.Bot.RpsTimerInterval))
	defer ticker.Stop()

	lastTradeCount := int64(0)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			total := atomic.LoadInt64(&a.tradeCount)
			diff := total - lastTradeCount

			rps := float64(diff) / float64(a.cfg.Exchange.Bot.RpsTimerInterval)

			go func() {
				a.logger.Info().
					Int64("total", total).
					Int64("diff", diff).
					Float64("rps", math.Round(rps)).
					Msg("trade count and rps")
			}()

			lastTradeCount = total
		}
	}
}

func (a *ArbitrageBot) checkBalances(ctx context.Context) {
	minBalance := decimal.NewFromInt(minBalanceForTrading)

	balanceStore := newBalanceStore(a.logger)
	balanceStore.Start(ctx, a.clients)

	cl := make([]exchange.Provider, 0, len(a.clients))

	for _, client := range a.clients {
		balances, ok := balanceStore.Get(client.GetExchangeName())

		if ok {
			fmt.Println(client.GetExchangeName())
			fmt.Printf("%+v\n", balances)
			for _, balance := range balances {
				if balance.Asset == "USDT" && balance.Free.GreaterThanOrEqual(minBalance) {
					cl = append(cl, client)
				}
			}
		}
	}

	if len(cl) < 2 {
		a.logger.Fatal().Msgf("not enough balance for trading")
	}

	for _, client := range cl {
		a.logger.Info().Msgf("enough balance for trading on %s", client.GetExchangeName())
	}

	a.clients = cl
}

func (a *ArbitrageBot) skipExchange() []exchange.Provider {
	skipSet := make(map[string]struct{}, len(a.cfg.Exchange.ArbitrageBot.SkipExchanges))
	for _, skip := range a.cfg.Exchange.ArbitrageBot.SkipExchanges {
		skipSet[strings.ToLower(skip)] = struct{}{}
	}

	restClients := make([]exchange.Provider, 0, len(a.clients))

	for _, client := range a.clients {
		_, ok := skipSet[strings.ToLower(client.GetExchangeName())]
		if !ok {
			restClients = append(restClients, client)
		} else {
			a.logger.Info().Msgf("Siletn mode: skip exchange %s", client.GetExchangeName())
		}
	}

	return restClients
}
