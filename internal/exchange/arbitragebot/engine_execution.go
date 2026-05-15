package arbitragebot

import (
	"context"
	"strconv"
	"sync"

	"github.com/shopspring/decimal"

	"github.com/lucrumx/bot/internal/exchange"
	"github.com/lucrumx/bot/internal/models"
)

// canBeOpened checks pre-conditions before opening a new arbitrage position.
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

// openPosition builds open orders for both legs (price comes from strategy) and dispatches them
// asynchronously via submitOpenLegs.
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
		// rawQty < step → the smallest tradable size on the combined exchanges costs more
		// than our notional. Log enough detail so the operator can decide whether to raise
		// notional or accept that this symbol is too thick.
		minNotional := step.Mul(decimal.NewFromFloat(event.BuyPrice))
		e.logger.Warn().
			Str("symbol", event.Symbol).
			Str("buy_on", event.BuyOnExchange).
			Str("sell_on", event.SellOnExchange).
			Stringer("notional_usdt", notional).
			Stringer("price", decimal.NewFromFloat(event.BuyPrice)).
			Stringer("raw_qty", rawQty).
			Stringer("coin_step", step).
			Stringer("min_notional_usdt", minNotional).
			Msg("⛔ execution: skipping — notional too small for combined coin step (raise notional or skip symbol)")
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
		OpenBuyLeg:   Leg{OrderID: buyOrder.ID},
		OpenSellLeg:  Leg{OrderID: sellOrder.ID},
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

// consumeExecutions reads execution events from a client channel and dispatches them to handleExecution.
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

// handleExecution routes a single execution event to the matching Position and applies the resulting transition.
func (e *Engine) handleExecution(ctx context.Context, event exchange.OrderExecutionEvent) {
	pos := e.pm.FindByOrderID(event.OrderID)
	if pos == nil {
		return
	}

	go e.markOrderFilled(ctx, event)

	state := pos.GetState()

	switch state {
	case PositionStateOpening, PositionStateOpeningPendingClose:
		if event.LeavesQty.IsPositive() {
			e.logger.Warn().
				Str("symbol", pos.Symbol).
				Str("order_id", event.OrderID.String()).
				Stringer("exec_qty", event.ExecQty).
				Stringer("leaves_qty", event.LeavesQty).
				Stringer("order_qty", event.OrderQty).
				Msg("⚠️ execution: PARTIAL FILL on open leg — model treats as FULLY confirmed, real position size is smaller")
		}
		e.logger.Info().
			Str("symbol", pos.Symbol).
			Str("order_id", event.OrderID.String()).
			Stringer("exec_qty", event.ExecQty).
			Msg("✅ execution: open leg confirmed")

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
		if event.LeavesQty.IsPositive() {
			e.logger.Warn().
				Str("symbol", pos.Symbol).
				Str("order_id", event.OrderID.String()).
				Stringer("exec_qty", event.ExecQty).
				Stringer("leaves_qty", event.LeavesQty).
				Stringer("order_qty", event.OrderQty).
				Msg("⚠️ execution: PARTIAL FILL on close leg — exchange may still have open exposure")
		}
		e.logger.Info().
			Str("symbol", pos.Symbol).
			Str("order_id", event.OrderID.String()).
			Stringer("exec_qty", event.ExecQty).
			Msg("✅ execution: close leg confirmed")

		transition, err := pos.OnCloseLegFilled(event.OrderID, event.ExecPrice, event.ExecQty)
		if err != nil {
			e.logger.Warn().Err(err).Msg("handleExecution: OnCloseLegFilled")
			return
		}

		e.applyTransition(ctx, pos, transition)

	case PositionStateOpen:
		// Position is fully open and waiting for the close signal. A fill event here usually means
		// the exchange retransmitted an open-leg event (or, very rarely, a partial-fill straggler
		// from a leg already marked Confirmed). markOrderFilled has already been called at the top
		// of this function — that update is idempotent on the same orderID, so there's nothing more
		// to do here. Logged at Debug so the operator can spot any unusual repeats without noise.
		e.logger.Debug().
			Str("symbol", pos.Symbol).
			Str("order_id", event.OrderID.String()).
			Msg("execution: stray fill event for already-open position (likely retransmit)")

	case PositionStateTimedOut:
		// Emergency cleanup fill (from watchFillTimeout or cleanupAfterPartnerFailed). markOrderFilled
		// has already been called at the top, so the DB row is updated with actual exec price/qty.
		// We just log and wait for the delayed pm.Delete to clean up.
		if event.LeavesQty.IsPositive() {
			e.logger.Warn().
				Str("symbol", pos.Symbol).
				Str("order_id", event.OrderID.String()).
				Stringer("exec_qty", event.ExecQty).
				Stringer("leaves_qty", event.LeavesQty).
				Stringer("order_qty", event.OrderQty).
				Msg("⚠️ execution: PARTIAL FILL on emergency close leg — verify exchange for residual position")
		}
		e.logger.Info().
			Str("symbol", pos.Symbol).
			Str("order_id", event.OrderID.String()).
			Stringer("exec_qty", event.ExecQty).
			Stringer("exec_price", event.ExecPrice).
			Msg("✅ execution: emergency close leg confirmed")
	}
}

// applyTransition executes the action required by a state transition returned from Position methods.
func (e *Engine) applyTransition(ctx context.Context, pos *Position, t PositionTransition) {
	switch t {
	case TransitionNone:
		// No action required — e.g. only one leg confirmed and we're waiting for the other.
		return
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

// submitOpenLegs concurrently sends both open orders to their exchanges. On error in either leg it
// blacklists the symbol, marks the spread Failed, and cleans up the partner via cleanupAfterPartnerFailed.
// On full success it spawns a fill-timeout watcher if the strategy requires one.
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

	// clean up whichever leg actually made it to the exchange:
	//  - for limit orders: CancelOrder (the order sits unfilled in the book, no position yet);
	//  - for market orders: CloseOrder (the order filled instantly, real position exists).
	if buyErr == nil {
		go e.cleanupAfterPartnerFailed(ctx, buyClient, pos, models.OrderSideBuy, pos.BuyExchange, buyOrder)
	}
	if sellErr == nil {
		go e.cleanupAfterPartnerFailed(ctx, sellClient, pos, models.OrderSideSell, pos.SellExchange, sellOrder)
	}
}

// cleanupAfterPartnerFailed is called when one leg's CreateOrder failed and we need to undo the
// other leg's order on the exchange.
//
// Three possible states for the surviving leg:
//  1. LIMIT, still unfilled in the book → CancelOrder is sufficient.
//  2. LIMIT, already filled (execution event arrived before us) → cancel would 404; market-close.
//  3. MARKET, instant fill → market-close.
//
// We detect (2) via pos.IsOpenLegConfirmed before attempting cancel, and as a fallback after
// cancel returns an error (the limit could fill in the small window between check and cancel).
func (e *Engine) cleanupAfterPartnerFailed(ctx context.Context, client exchange.Provider, pos *Position, side models.OrderSide, exchangeName string, order models.Order) {
	if order.Type == models.OrderTypeLimit {
		// (2) already filled — skip the doomed cancel, go straight to emergency close.
		if pos.IsOpenLegConfirmed(side) {
			e.logger.Warn().
				Str("symbol", pos.Symbol).
				Str("exchange", exchangeName).
				Str("side", string(side)).
				Msg("partner-failed: limit already filled before cleanup — falling back to emergency market close")
			e.submitEmergencyClose(ctx, client, pos, side, exchangeName, "partner-failed: filled limit leg emergency-closed")
			return
		}

		// (1) still unfilled — try cancel.
		if err := client.CancelOrder(ctx, order.ID, order.ExchangeOrderID, pos.Symbol); err != nil {
			// Race: limit may have filled between the IsOpenLegConfirmed check and the cancel
			// call. Re-check confirmation; if now true, fall through to emergency close.
			if pos.IsOpenLegConfirmed(side) {
				e.logger.Warn().
					Err(err).
					Str("symbol", pos.Symbol).
					Str("exchange", exchangeName).
					Str("side", string(side)).
					Msg("partner-failed: cancel returned error but leg filled in the meantime — falling back to emergency close")
				e.submitEmergencyClose(ctx, client, pos, side, exchangeName, "partner-failed: filled limit leg emergency-closed (after cancel race)")
				return
			}
			e.logger.Error().
				Err(err).
				Str("symbol", pos.Symbol).
				Str("exchange", exchangeName).
				Str("side", string(side)).
				Msg("⚠️ partner-failed: cancel of posted limit failed and leg is NOT confirmed — VERIFY EXCHANGE for stuck order or open position")
			return
		}
		e.markOrderCanceled(ctx, order.ID)
		e.logger.Info().
			Str("symbol", pos.Symbol).
			Str("exchange", exchangeName).
			Str("side", string(side)).
			Msg("✅ partner-failed: unfilled limit cancelled cleanly")
		return
	}

	// (3) market order: real position exists, close it with reduce-only market.
	e.submitEmergencyClose(ctx, client, pos, side, exchangeName, "partner-failed: filled market leg emergency-closed")
}

// submitCloseLegs concurrently sends close orders for both legs of an open position.
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

// emergencyClose is the action for TransitionEmergencyClose — same cleanup as fill timeout with no legs filled.
func (e *Engine) emergencyClose(ctx context.Context, pos *Position) {
	e.logger.Warn().Str("symbol", pos.Symbol).Msg("execution: emergency close")
	e.pm.Delete(pos)
	go e.markSpreadFailed(ctx, pos)
}
