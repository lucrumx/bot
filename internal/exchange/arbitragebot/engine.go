package arbitragebot

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"

	"github.com/lucrumx/bot/internal/config"
	"github.com/lucrumx/bot/internal/exchange"
	"github.com/lucrumx/bot/internal/models"
	"github.com/lucrumx/bot/internal/notifier"
	"github.com/lucrumx/bot/internal/utils"
)

// Engine is a thin orchestrator: receives spread signals, routes execution events
// to the correct Position, and applies state transitions.
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
	e.clients = make(map[string]exchange.Provider, len(clients))
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

// CommonSymbols returns symbols available on all active exchanges.
func (e *Engine) CommonSymbols() map[string]struct{} {
	result := map[string]struct{}{}
	first := true

	for exchangeName := range e.clients {
		instruments := e.instruments[exchangeName]
		if first {
			for symbol := range instruments {
				result[symbol] = struct{}{}
			}
			first = false
			continue
		}
		for s := range result {
			if _, ok := instruments[s]; !ok {
				delete(result, s)
			}
		}
	}

	return result
}

// --- signal handling ---

func (e *Engine) handleOpen(ctx context.Context, event *SpreadEvent) {
	if e.pm.IsBlacklisted(event.Symbol) {
		return
	}

	go e.sendOpenNotification(event)

	if e.isSilentMode() {
		go e.saveSpread(ctx, event, nil)
		return
	}

	pos := e.openPosition(ctx, event)
	go e.saveSpread(ctx, event, pos)
}

func (e *Engine) handleUpdate(ctx context.Context, event *SpreadEvent) {
	go func() {
		spreadStr := strconv.FormatFloat(event.MaxSpreadPercent, 'f', 2, 64)
		e.logger.Warn().
			Str("pair", event.Symbol).
			Str("spread", spreadStr).
			Str("buy on", event.BuyOnExchange).
			Str("sell on", event.SellOnExchange).
			Msg("🔥 SPREAD GROWING")

		msg := fmt.Sprintf("<b>🔔 ARBITRAGE: Ticker - %s</b>\n\nSpread is growing, new spread: <code>%s%%</code>",
			event.Symbol, spreadStr)
		if err := e.notif.Send(msg); err != nil {
			e.logger.Warn().Err(err).Msg("failed to send telegram notification")
		}
	}()

	go func() {
		err := e.spreadRepo.Update(ctx, &models.ArbitrageSpread{
			Status:           models.ArbitrageSpreadUpdated,
			MaxSpreadPercent: decimal.NewFromFloat(event.MaxSpreadPercent),
			UpdatedAt:        time.Now(),
		}, FindFilter{
			Symbol: event.Symbol,
			BuyEx:  event.BuyOnExchange,
			SellEx: event.SellOnExchange,
			Status: []models.ArbitrageSpreadStatus{models.ArbitrageSpreadOpened, models.ArbitrageSpreadUpdated},
		})
		if err != nil {
			e.logger.Warn().Err(err).Msgf("failed to update spread symbol=%s", event.Symbol)
		}
	}()
}

func (e *Engine) handleClose(ctx context.Context, event *SpreadEvent) {
	go func() {
		e.logger.Warn().Str("pair", event.Symbol).Msg("🔥 SPREAD CLOSED")
		msg := fmt.Sprintf("<b>🔔 ARBITRAGE: Ticker - %s</b>\n\nSpread closed", event.Symbol)
		if err := e.notif.Send(msg); err != nil {
			e.logger.Warn().Err(err).Msg("failed to send telegram notification")
		}
	}()

	if !e.isSilentMode() {
		pos := e.pm.FindByKey(event.Symbol, event.BuyOnExchange, event.SellOnExchange)
		if pos != nil {
			transition := pos.RequestClose()
			e.applyTransition(ctx, pos, transition)
		}
	}

	go func() {
		err := e.spreadRepo.Update(ctx, &models.ArbitrageSpread{
			Status:    models.ArbitrageSpreadClosed,
			UpdatedAt: time.Now(),
		}, FindFilter{
			Symbol: event.Symbol,
			BuyEx:  event.BuyOnExchange,
			SellEx: event.SellOnExchange,
			Status: []models.ArbitrageSpreadStatus{models.ArbitrageSpreadOpened, models.ArbitrageSpreadUpdated},
		})
		if err != nil {
			e.logger.Warn().Err(err).Msgf("failed to update spread symbol=%s", event.Symbol)
		}
	}()
}

// --- position lifecycle ---

func (e *Engine) canBeOpened(event *SpreadEvent) bool {
	if e.pm.Count() >= 3 {
		e.logger.Info().Msg("execution: already have 3 open positions, skipping")
		return false
	}

	if event.FromSpreadPercent > e.cfg.Exchange.ArbitrageBot.MaxSpreadPercentForOpen {
		e.logger.Info().
			Str("spread", strconv.FormatFloat(event.FromSpreadPercent, 'f', 2, 64)).
			Str("symbol", event.Symbol).
			Msg("execution: spread is too high, skipping")
		return false
	}

	if e.pm.HasOverlap(event.Symbol, event.BuyOnExchange, event.SellOnExchange) {
		e.logger.Debug().
			Str("symbol", event.Symbol).
			Str("buy_on", event.BuyOnExchange).
			Str("sell_on", event.SellOnExchange).
			Msg("execution: symbol already open on one of the exchanges, skipping")
		return false
	}

	if event.BuyPrice == 0 {
		e.logger.Error().Str("symbol", event.Symbol).Msg("execution: buy price is zero")
		return false
	}

	return true
}

func (e *Engine) openPosition(ctx context.Context, event *SpreadEvent) *Position {
	if !e.canBeOpened(event) {
		return nil
	}

	step, err := e.coinStep(event.Symbol, event.BuyOnExchange, event.SellOnExchange)
	if err != nil {
		e.logger.Error().Err(err).Str("symbol", event.Symbol).Msg("execution: failed to get coin step")
		return nil
	}

	notional := decimal.NewFromInt(e.notional())
	rawQty := notional.Div(decimal.NewFromFloat(event.BuyPrice))
	qty := rawQty.Div(step).Floor().Mul(step)

	if qty.IsZero() {
		e.logger.Error().Str("symbol", event.Symbol).Msg("execution: aligned qty is zero")
		return nil
	}

	buyVol, err := e.qtyForExchange(qty, event.Symbol, event.BuyOnExchange)
	if err != nil {
		e.logger.Error().Err(err).Str("symbol", event.Symbol).Msg("execution: failed to convert buy qty")
		return nil
	}

	sellVol, err := e.qtyForExchange(qty, event.Symbol, event.SellOnExchange)
	if err != nil {
		e.logger.Error().Err(err).Str("symbol", event.Symbol).Msg("execution: failed to convert sell qty")
		return nil
	}

	buyOrder, err := e.buildOrder(event.Symbol, models.OrderSideBuy, buyVol, event.BuyOnExchange,
		e.strategy.OpenPrice(event, models.OrderSideBuy))
	if err != nil {
		e.logger.Error().Err(err).Str("symbol", event.Symbol).Msg("execution: failed to build buy order")
		return nil
	}

	sellOrder, err := e.buildOrder(event.Symbol, models.OrderSideSell, sellVol, event.SellOnExchange,
		e.strategy.OpenPrice(event, models.OrderSideSell))
	if err != nil {
		e.logger.Error().Err(err).Str("symbol", event.Symbol).Msg("execution: failed to build sell order")
		return nil
	}

	pos := &Position{
		Symbol:       event.Symbol,
		BuyExchange:  event.BuyOnExchange,
		SellExchange: event.SellOnExchange,
		QtyCoins:     qty,
		OpenBuyLeg:  Leg{OrderID: buyOrder.ID},
		OpenSellLeg: Leg{OrderID: sellOrder.ID},
		State:        PositionStateOpening,
	}

	e.pm.Add(pos)

	e.logger.Info().
		Str("symbol", event.Symbol).
		Str("buy_on", event.BuyOnExchange).
		Str("sell_on", event.SellOnExchange).
		Stringer("qty_coins", qty).
		Stringer("buy_vol", buyVol).
		Stringer("sell_vol", sellVol).
		Msg("🚀 execution: opening position")

	go e.submitOpenLegs(ctx, pos, buyOrder, sellOrder)

	return pos
}

// --- execution routing ---

func (e *Engine) consumeExecutions(ctx context.Context, ch <-chan exchange.OrderExecutionEvent) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			e.handleExecution(ctx, event)
		}
	}
}

func (e *Engine) handleExecution(ctx context.Context, event exchange.OrderExecutionEvent) {
	pos := e.pm.FindByOrderID(event.OrderID)
	if pos == nil {
		return
	}

	go e.markOrderFilled(ctx, event)

	state := pos.GetState()

	switch state {
	case PositionStateOpening, PositionStateOpeningPendingClose:
		e.logger.Info().Str("symbol", pos.Symbol).Str("order_id", event.OrderID.String()).Msg("✅ execution: open leg confirmed")

		transition, err := pos.OnOpenLegFilled(event.OrderID, event.ExecPrice, event.ExecQty)
		if err != nil {
			e.logger.Warn().Err(err).Msg("handleExecution: OnOpenLegFilled")
			return
		}

		if transition == TransitionNone && pos.GetState() == PositionStateOpen {
			e.logger.Info().Str("symbol", pos.Symbol).Msg("✅ execution: position open")
		}

		e.applyTransition(ctx, pos, transition)

	case PositionStateClosing:
		e.logger.Info().Str("symbol", pos.Symbol).Str("order_id", event.OrderID.String()).Msg("✅ execution: close leg confirmed")

		transition, err := pos.OnCloseLegFilled(event.OrderID, event.ExecPrice, event.ExecQty)
		if err != nil {
			e.logger.Warn().Err(err).Msg("handleExecution: OnCloseLegFilled")
			return
		}

		e.applyTransition(ctx, pos, transition)
	}
}

// applyTransition executes the action required by a state transition.
func (e *Engine) applyTransition(ctx context.Context, pos *Position, t PositionTransition) {
	switch t {
	case TransitionSubmitClose:
		e.logger.Info().Str("symbol", pos.Symbol).Msg("⚡ execution: submitting close legs")
		go e.submitCloseLegs(ctx, pos)
	case TransitionEmergencyClose:
		go e.emergencyClose(ctx, pos)
	case TransitionFullyClosed:
		e.logger.Info().Str("symbol", pos.Symbol).Msg("💰 execution: position fully closed")
		e.pm.Delete(pos)
	}
}

// --- order submission ---

func (e *Engine) submitOpenLegs(ctx context.Context, pos *Position, buyOrder, sellOrder models.Order) {
	buyClient := e.clients[pos.BuyExchange]
	sellClient := e.clients[pos.SellExchange]

	var wg sync.WaitGroup
	wg.Add(2)

	var buyErr, sellErr error

	// save orders before CreateOrder so execution events can always find them in DB
	e.saveOrder(&buyOrder)
	e.saveOrder(&sellOrder)

	go func() {
		defer wg.Done()
		buyErr = buyClient.CreateOrder(ctx, &buyOrder)
		if buyErr != nil {
			// order was saved before CreateOrder — mark it rejected so DB reflects reality
			e.markOrderRejected(ctx, buyOrder.ID)
		} else {
			// store exchange order ID for cancel support (needed by MEXC)
			pos.SetOpenLegExchangeOrderID(buyOrder.ID, buyOrder.ExchangeOrderID)
		}
	}()

	go func() {
		defer wg.Done()
		sellErr = sellClient.CreateOrder(ctx, &sellOrder)
		if sellErr != nil {
			e.markOrderRejected(ctx, sellOrder.ID)
		} else {
			pos.SetOpenLegExchangeOrderID(sellOrder.ID, sellOrder.ExchangeOrderID)
		}
	}()

	wg.Wait()

	if buyErr == nil && sellErr == nil {
		// both legs submitted — start fill timeout watcher for limit orders
		if timeout := e.strategy.FillTimeout(); timeout > 0 {
			go e.watchFillTimeout(ctx, pos, timeout)
		}
		return
	}

	e.logger.Error().
		AnErr("buy_err", buyErr).
		AnErr("sell_err", sellErr).
		Str("symbol", pos.Symbol).
		Msg("execution: failed to submit open legs")

	e.pm.Blacklist(pos)
	e.logger.Warn().Str("symbol", pos.Symbol).Msg("execution: symbol blacklisted after failed open")

	go e.markSpreadFailed(ctx, pos)

	// emergency close whichever leg succeeded
	if buyErr == nil {
		go func() {
			closeOrder := buyOrder
			closeOrder.ID = uuid.New()
			if err := buyClient.CloseOrder(ctx, &closeOrder); err != nil {
				e.logger.Error().Err(err).Str("symbol", pos.Symbol).Msg("execution: emergency close buy leg failed")
			}
		}()
	}
	if sellErr == nil {
		go func() {
			closeOrder := sellOrder
			closeOrder.ID = uuid.New()
			if err := sellClient.CloseOrder(ctx, &closeOrder); err != nil {
				e.logger.Error().Err(err).Str("symbol", pos.Symbol).Msg("execution: emergency close sell leg failed")
			}
		}()
	}
}

func (e *Engine) submitCloseLegs(ctx context.Context, pos *Position) {
	buyClient := e.clients[pos.BuyExchange]
	sellClient := e.clients[pos.SellExchange]

	// helper: on any preparation error, delete the position so the slot is freed
	fail := func(msg string, err error) {
		e.logger.Error().Err(err).Str("symbol", pos.Symbol).Msg(msg)
		e.pm.Delete(pos)
	}

	buyVol, err := e.qtyForExchange(pos.QtyCoins, pos.Symbol, pos.BuyExchange)
	if err != nil {
		fail("execution: failed to convert close buy qty", err)
		return
	}
	sellVol, err := e.qtyForExchange(pos.QtyCoins, pos.Symbol, pos.SellExchange)
	if err != nil {
		fail("execution: failed to convert close sell qty", err)
		return
	}

	buyOrder, err := e.buildOrder(pos.Symbol, models.OrderSideBuy, buyVol, pos.BuyExchange,
		e.strategy.ClosePrice(pos, models.OrderSideBuy))
	if err != nil {
		fail("execution: failed to build close buy order", err)
		return
	}
	sellOrder, err := e.buildOrder(pos.Symbol, models.OrderSideSell, sellVol, pos.SellExchange,
		e.strategy.ClosePrice(pos, models.OrderSideSell))
	if err != nil {
		fail("execution: failed to build close sell order", err)
		return
	}

	// register close leg IDs before submitting so execution events can be matched
	closeBuyLeg := Leg{OrderID: buyOrder.ID}
	closeSellLeg := Leg{OrderID: sellOrder.ID}
	pos.SetCloseLegIDs(closeBuyLeg, closeSellLeg)
	e.pm.IndexCloseLeg(pos, buyOrder.ID, sellOrder.ID)

	// save orders before CloseOrder so execution events can always find them in DB
	e.saveOrder(&buyOrder)
	e.saveOrder(&sellOrder)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		if err := buyClient.CloseOrder(ctx, &buyOrder); err != nil {
			e.logger.Error().Err(err).Str("symbol", pos.Symbol).Str("exchange", pos.BuyExchange).Msg("execution: failed to close buy leg")
		}
	}()

	go func() {
		defer wg.Done()
		if err := sellClient.CloseOrder(ctx, &sellOrder); err != nil {
			e.logger.Error().Err(err).Str("symbol", pos.Symbol).Str("exchange", pos.SellExchange).Msg("execution: failed to close sell leg")
		}
	}()

	wg.Wait()

	e.logger.Info().Str("symbol", pos.Symbol).Msg("🔻 execution: close orders submitted, waiting for confirmations")

	go e.saveCloseOrderIDsToSpread(ctx, pos, buyOrder.ID, sellOrder.ID)
}

func (e *Engine) emergencyClose(ctx context.Context, pos *Position) {
	// used by TransitionEmergencyClose — same cleanup as timeout with no legs filled
	e.logger.Warn().Str("symbol", pos.Symbol).Msg("execution: emergency close")
	e.pm.Delete(pos)
	go e.markSpreadFailed(ctx, pos)
}

// watchFillTimeout waits for the fill timeout and triggers cleanup if legs haven't filled.
func (e *Engine) watchFillTimeout(ctx context.Context, pos *Position, timeout time.Duration) {
	select {
	case <-ctx.Done():
		return
	case <-time.After(timeout):
	}

	shouldAct, info := pos.OnOpenTimeout()
	if !shouldAct {
		return // position already transitioned normally
	}

	e.logger.Warn().
		Str("symbol", pos.Symbol).
		Bool("buy_filled", info.BuyFilled).
		Bool("sell_filled", info.SellFilled).
		Msgf("⏱ execution: fill timeout after %s", timeout)

	e.pm.Delete(pos)
	e.handleFillTimeout(ctx, pos, info)
}

// handleFillTimeout cancels pending legs and emergency-closes any filled legs.
func (e *Engine) handleFillTimeout(ctx context.Context, pos *Position, info OpenTimeoutInfo) {
	buyClient := e.clients[pos.BuyExchange]
	sellClient := e.clients[pos.SellExchange]

	// cancel whichever leg is still pending
	if !info.BuyFilled {
		go func() {
			if err := buyClient.CancelOrder(ctx, info.BuyOrderID, info.BuyExchangeOrderID, pos.Symbol); err != nil {
				e.logger.Error().Err(err).Str("symbol", pos.Symbol).Str("exchange", pos.BuyExchange).
					Msg("timeout: failed to cancel pending buy leg")
			}
		}()
	}
	if !info.SellFilled {
		go func() {
			if err := sellClient.CancelOrder(ctx, info.SellOrderID, info.SellExchangeOrderID, pos.Symbol); err != nil {
				e.logger.Error().Err(err).Str("symbol", pos.Symbol).Str("exchange", pos.SellExchange).
					Msg("timeout: failed to cancel pending sell leg")
			}
		}()
	}

	// market-close whichever leg was already filled to neutralize the position
	if info.BuyFilled {
		go func() {
			buyVol, err := e.qtyForExchange(pos.QtyCoins, pos.Symbol, pos.BuyExchange)
			if err != nil {
				e.logger.Error().Err(err).Str("symbol", pos.Symbol).Msg("timeout: failed to get buy vol for emergency close")
				return
			}
			closeOrder, err := e.buildOrder(pos.Symbol, models.OrderSideBuy, buyVol, pos.BuyExchange, nil)
			if err != nil {
				e.logger.Error().Err(err).Str("symbol", pos.Symbol).Msg("timeout: failed to build emergency close buy order")
				return
			}
			closeOrder.ID = uuid.New()
			if err := buyClient.CloseOrder(ctx, &closeOrder); err != nil {
				e.logger.Error().Err(err).Str("symbol", pos.Symbol).Str("exchange", pos.BuyExchange).
					Msg("timeout: failed to emergency close filled buy leg")
			}
		}()
	}
	if info.SellFilled {
		go func() {
			sellVol, err := e.qtyForExchange(pos.QtyCoins, pos.Symbol, pos.SellExchange)
			if err != nil {
				e.logger.Error().Err(err).Str("symbol", pos.Symbol).Msg("timeout: failed to get sell vol for emergency close")
				return
			}
			closeOrder, err := e.buildOrder(pos.Symbol, models.OrderSideSell, sellVol, pos.SellExchange, nil)
			if err != nil {
				e.logger.Error().Err(err).Str("symbol", pos.Symbol).Msg("timeout: failed to build emergency close sell order")
				return
			}
			closeOrder.ID = uuid.New()
			if err := sellClient.CloseOrder(ctx, &closeOrder); err != nil {
				e.logger.Error().Err(err).Str("symbol", pos.Symbol).Str("exchange", pos.SellExchange).
					Msg("timeout: failed to emergency close filled sell leg")
			}
		}()
	}

	go e.markSpreadFailed(ctx, pos)
}

// --- persistence helpers ---

func (e *Engine) saveSpread(ctx context.Context, event *SpreadEvent, pos *Position) {
	spread := &models.ArbitrageSpread{
		Symbol:           event.Symbol,
		BuyOnExchange:    event.BuyOnExchange,
		SellOnExchange:   event.SellOnExchange,
		BuyPrice:         decimal.NewFromFloat(event.BuyPrice),
		SellPrice:        decimal.NewFromFloat(event.SellPrice),
		SpreadPercent:    decimal.NewFromFloat(event.FromSpreadPercent),
		MaxSpreadPercent: decimal.NewFromFloat(event.MaxSpreadPercent),
		Status:           event.Status,
	}
	if pos != nil {
		spread.OpenBuyOrderID = pos.OpenBuyLeg.OrderID
		spread.OpenSellOrderID = pos.OpenSellLeg.OrderID
	}
	if err := e.spreadRepo.Create(ctx, spread); err != nil {
		e.logger.Warn().Err(err).Msgf("failed to create spread symbol=%s", event.Symbol)
	}
}

func (e *Engine) saveCloseOrderIDsToSpread(ctx context.Context, pos *Position, closeBuyID, closeSellID uuid.UUID) {
	spread, err := e.spreadRepo.FindOne(ctx, FindFilter{
		Symbol: pos.Symbol,
		BuyEx:  pos.BuyExchange,
		SellEx: pos.SellExchange,
		Status: []models.ArbitrageSpreadStatus{models.ArbitrageSpreadOpened, models.ArbitrageSpreadUpdated, models.ArbitrageSpreadClosed},
	})
	if err != nil {
		e.logger.Warn().Err(err).Msgf("failed to find spread for close order update symbol=%s", pos.Symbol)
		return
	}

	if spread.CloseBuyOrderID != uuid.Nil || spread.CloseSellOrderID != uuid.Nil {
		e.logger.Warn().Msgf("spread already has close order IDs, skipping symbol=%s", pos.Symbol)
		return
	}

	err = e.spreadRepo.Update(ctx, &models.ArbitrageSpread{
		CloseBuyOrderID:  closeBuyID,
		CloseSellOrderID: closeSellID,
		UpdatedAt:        time.Now(),
	}, FindFilter{ID: spread.ID})
	if err != nil {
		e.logger.Warn().Err(err).Msgf("failed to update spread with close order IDs symbol=%s", pos.Symbol)
	}
}

func (e *Engine) markSpreadFailed(ctx context.Context, pos *Position) {
	err := e.spreadRepo.Update(ctx, &models.ArbitrageSpread{
		Status:    models.ArbitrageSpreadFailed,
		UpdatedAt: time.Now(),
	}, FindFilter{
		Symbol: pos.Symbol,
		BuyEx:  pos.BuyExchange,
		SellEx: pos.SellExchange,
		Status: []models.ArbitrageSpreadStatus{models.ArbitrageSpreadOpened},
	})
	if err != nil {
		e.logger.Warn().Err(err).Str("symbol", pos.Symbol).Msg("failed to mark spread as FAILED")
	}
}

func (e *Engine) saveOrder(order *models.Order) {
	if e.orderRepo == nil {
		return
	}
	if err := e.orderRepo.Create(context.Background(), order); err != nil {
		e.logger.Error().Err(err).Str("order_id", order.ID.String()).Msg("execution: failed to save order")
	}
}

func (e *Engine) markOrderRejected(ctx context.Context, orderID uuid.UUID) {
	if e.orderRepo == nil {
		return
	}
	rejected := models.OrderStatusRejected
	if err := e.orderRepo.UpdatePartialy(ctx, orderID, OrderPatch{Status: &rejected}); err != nil {
		e.logger.Error().Err(err).Str("order_id", orderID.String()).Msg("execution: failed to mark order rejected")
	}
}

func (e *Engine) markOrderFilled(ctx context.Context, event exchange.OrderExecutionEvent) {
	if e.orderRepo == nil {
		return
	}
	if err := e.orderRepo.UpdateFilled(ctx, event.OrderID, event.ExecPrice, event.ExecQty); err != nil {
		e.logger.Error().Err(err).Str("order_id", event.OrderID.String()).Msg("execution: failed to mark order filled")
	}
}

// --- notifications ---

func (e *Engine) sendOpenNotification(event *SpreadEvent) {
	spreadStr := strconv.FormatFloat(event.FromSpreadPercent, 'f', 2, 64)
	e.logger.Warn().
		Str("pair", event.Symbol).
		Str("spread", spreadStr).
		Str("buy on", event.BuyOnExchange).
		Str("sell on", event.SellOnExchange).
		Msg("🔥 SPREAD DETECTED")

	msg := fmt.Sprintf(
		"<b>🔔 ARBITRAGE: Ticker - %s</b>\n\n"+
			"Spread: <code>%s%%</code>\n\n"+
			"🟢 Buy:  %s - <b>%s</b>\n"+
			"🔴 Sell: %s - <b>%s</b>",
		event.Symbol, spreadStr,
		event.BuyOnExchange, utils.FormatPrice(event.BuyPrice),
		event.SellOnExchange, utils.FormatPrice(event.SellPrice),
	)
	if err := e.notif.Send(msg); err != nil {
		e.logger.Warn().Err(err).Msg("failed to send telegram notification")
	}
}

// --- instrument helpers ---

func (e *Engine) coinStep(symbol, exchange1, exchange2 string) (decimal.Decimal, error) {
	inst1, err := e.instrumentFor(symbol, exchange1)
	if err != nil {
		return decimal.Zero, err
	}
	inst2, err := e.instrumentFor(symbol, exchange2)
	if err != nil {
		return decimal.Zero, err
	}
	return lcmDecimal(inst1.VolStep.Mul(inst1.ContractSize), inst2.VolStep.Mul(inst2.ContractSize)), nil
}

func (e *Engine) instrumentFor(symbol, exchangeName string) (exchange.Instrument, error) {
	instruments, ok := e.instruments[exchangeName]
	if !ok {
		return exchange.Instrument{}, fmt.Errorf("no instrument data for exchange %s", exchangeName)
	}
	inst, ok := instruments[symbol]
	if !ok {
		return exchange.Instrument{}, fmt.Errorf("no instrument %s on %s", symbol, exchangeName)
	}
	return inst, nil
}

func (e *Engine) qtyForExchange(qty decimal.Decimal, symbol, exchangeName string) (decimal.Decimal, error) {
	inst, err := e.instrumentFor(symbol, exchangeName)
	if err != nil {
		return decimal.Zero, err
	}
	return qty.Div(inst.ContractSize), nil
}

func (e *Engine) buildOrder(symbol string, side models.OrderSide, qty decimal.Decimal, exchangeName string, price *decimal.Decimal) (models.Order, error) {
	orderType := models.OrderTypeMarket
	if price != nil {
		orderType = models.OrderTypeLimit
	}
	dto := exchange.CreateOrderDto{
		Symbol:       symbol,
		Side:         side,
		Type:         orderType,
		Market:       models.OrderMarketLinear,
		Quantity:     qty,
		ExchangeName: exchangeName,
	}
	if price != nil {
		dto.Price = *price
	}
	return exchange.MakeOrderStruct(dto)
}

func (e *Engine) notional() int64 {
	return 10
}

func (e *Engine) isSilentMode() bool {
	return e.cfg.Exchange.ArbitrageBot.SilentMode
}

// --- math helpers ---

func gcdDecimal(a, b decimal.Decimal) decimal.Decimal {
	for !b.IsZero() {
		a, b = b, a.Mod(b)
	}
	return a
}

func lcmDecimal(a, b decimal.Decimal) decimal.Decimal {
	return a.Mul(b).Div(gcdDecimal(a, b))
}
