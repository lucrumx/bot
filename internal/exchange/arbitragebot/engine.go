package arbitragebot

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/lucrumx/bot/internal/config"
	"github.com/lucrumx/bot/internal/exchange"
	"github.com/lucrumx/bot/internal/models"
	"github.com/lucrumx/bot/internal/notifier"
)

// Engine is a thin orchestrator: receives spread signals, routes execution events to the correct
// Position, and applies state transitions. Each major responsibility lives in its own engine_*.go:
//
//	engine_signals.go      — spread event handlers (handleOpen / handleUpdate / handleClose)
//	engine_execution.go    — order submission and execution event routing
//	engine_fill_timeout.go — limit fill timeout watcher and cleanup
//	engine_persistence.go  — DB writes for spreads and orders
//	engine_helpers.go      — instrument math, order construction, price alignment
type Engine struct {
	cfg         *config.Config
	clients     map[string]exchange.Provider
	instruments map[string]map[string]exchange.Instrument // [exchange][symbol]
	orderRepo   OrderRepository
	spreadRepo  ArbitrageSpreadRepository
	notif       notifier.Notifier
	logger      zerolog.Logger
	strategy    OrderStrategy

	signalCh chan *SpreadEvent
	pm       *PositionManager
}

// NewEngine creates a new Engine with the given order strategy.
func NewEngine(
	cfg *config.Config,
	clients []exchange.Provider,
	orderRepo OrderRepository,
	spreadRepo ArbitrageSpreadRepository,
	notif notifier.Notifier,
	logger zerolog.Logger,
	strategy OrderStrategy,
) *Engine {
	clientMap := make(map[string]exchange.Provider, len(clients))
	for _, c := range clients {
		clientMap[c.GetExchangeName()] = c
	}
	return &Engine{
		cfg:         cfg,
		clients:     clientMap,
		instruments: make(map[string]map[string]exchange.Instrument),
		orderRepo:   orderRepo,
		spreadRepo:  spreadRepo,
		notif:       notif,
		logger:      logger,
		strategy:    strategy,
		signalCh:    make(chan *SpreadEvent, 1000),
		pm:          newPositionManager(),
	}
}

// LoadInstruments fetches contract specifications from all exchanges.
func (e *Engine) LoadInstruments(ctx context.Context, clients []exchange.Provider) error {
	for _, client := range clients {
		e.clients[client.GetExchangeName()] = client

		instruments, err := client.GetInstruments(ctx)
		if err != nil {
			return fmt.Errorf("failed to load instruments from %s: %w", client.GetExchangeName(), err)
		}
		e.instruments[client.GetExchangeName()] = instruments
	}
	return nil
}

// Instruments returns the cached instrument data per exchange (exchange → symbol → Instrument).
func (e *Engine) Instruments() map[string]map[string]exchange.Instrument {
	return e.instruments
}

// ListenExecutions subscribes to execution events from all clients.
func (e *Engine) ListenExecutions(ctx context.Context) error {
	for _, client := range e.clients {
		ch, err := client.SubscribeExecutions(ctx)
		if err != nil {
			return err
		}
		if ch == nil {
			e.logger.Warn().Str("exchange", client.GetExchangeName()).Msg("execution: no execution channel, skipping")
			continue
		}
		e.logger.Info().Str("exchange", client.GetExchangeName()).Msg("execution: listening for executions")
		go e.consumeExecutions(ctx, ch)
	}
	return nil
}

// Run processes spread signals from the channel.
func (e *Engine) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-e.signalCh:
			switch event.Status {
			case models.ArbitrageSpreadOpened:
				e.handleOpen(ctx, event)
			case models.ArbitrageSpreadUpdated:
				e.handleUpdate(ctx, event)
			case models.ArbitrageSpreadClosed:
				e.handleClose(ctx, event)
			}
		}
	}
}

// HandleSignal enqueues spread events for processing.
func (e *Engine) HandleSignal(events []*SpreadEvent) {
	for _, ev := range events {
		e.signalCh <- ev
	}
}
