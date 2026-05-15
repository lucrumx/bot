package arbitragebot

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/lucrumx/bot/internal/exchange"
	"github.com/lucrumx/bot/internal/models"
)

// emergencyCloseTrackingWindow is the safety window after which a TimedOut position is forcibly
// deleted from PositionManager, even if execution events for its emergency-close orders never
// arrived. Long enough for ByBit/MEXC to send fills via WS in normal conditions.
const emergencyCloseTrackingWindow = 10 * time.Second

// watchFillTimeout waits for the fill timeout and triggers cleanup if legs haven't filled.
// Atomicity is guaranteed by Position.OnOpenTimeout — it transitions the position to TimedOut
// under the position's mutex and returns false on subsequent calls, so this watcher won't race
// with concurrent fill events.
//
// IMPORTANT: we deliberately do NOT pm.Delete(pos) here. The handleFillTimeout call below may
// spawn emergency-close orders whose execution events need to route back through pm.byOrderID
// to update DB rows. Deletion is deferred to scheduleDelayedDelete.
func (e *Engine) watchFillTimeout(ctx context.Context, pos *Position, timeout time.Duration) {
	select {
	case <-ctx.Done():
		return
	case <-time.After(timeout):
	}

	shouldAct, info := pos.OnOpenTimeout()
	if !shouldAct {
		return // position already transitioned normally (e.g. both legs filled in time)
	}

	e.logger.Warn().
		Str("symbol", pos.Symbol).
		Bool("buy_filled", info.BuyFilled).
		Bool("sell_filled", info.SellFilled).
		Msgf("⏱ execution: fill timeout after %s", timeout)

	e.handleFillTimeout(ctx, pos, info)
	e.scheduleDelayedDelete(ctx, pos)
}

// scheduleDelayedDelete deletes pos from PositionManager after emergencyCloseTrackingWindow.
// This window gives execution events for emergency-close orders time to arrive and update DB
// before the orderID → pos mapping is torn down.
func (e *Engine) scheduleDelayedDelete(ctx context.Context, pos *Position) {
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(emergencyCloseTrackingWindow):
		}
		e.pm.Delete(pos)
	}()
}

// handleFillTimeout cancels still-pending legs and emergency-closes any filled legs.
func (e *Engine) handleFillTimeout(ctx context.Context, pos *Position, info OpenTimeoutInfo) {
	buyClient := e.clients[pos.BuyExchange]
	sellClient := e.clients[pos.SellExchange]

	// cancel whichever leg is still pending
	if !info.BuyFilled {
		go e.cancelPendingLeg(ctx, buyClient, pos.Symbol, pos.BuyExchange, models.OrderSideBuy, info.BuyOrderID, info.BuyExchangeOrderID)
	}
	if !info.SellFilled {
		go e.cancelPendingLeg(ctx, sellClient, pos.Symbol, pos.SellExchange, models.OrderSideSell, info.SellOrderID, info.SellExchangeOrderID)
	}

	// market-close whichever leg was already filled to neutralize the position
	if info.BuyFilled {
		go e.emergencyCloseLeg(ctx, buyClient, pos, models.OrderSideBuy, pos.BuyExchange)
	}
	if info.SellFilled {
		go e.emergencyCloseLeg(ctx, sellClient, pos, models.OrderSideSell, pos.SellExchange)
	}

	go e.markSpreadFailed(ctx, pos)
}

// cancelPendingLeg cancels an unfilled limit leg on the exchange and marks the order CANCELED in DB.
// On failure (e.g. order already filled or already cancelled), logs a warning — the order's DB row
// stays at its previous status and the operator may need to verify on the exchange manually.
func (e *Engine) cancelPendingLeg(ctx context.Context, client exchange.Provider, symbol, exchangeName string, side models.OrderSide, orderID uuid.UUID, exchangeOrderID string) {
	if err := client.CancelOrder(ctx, orderID, exchangeOrderID, symbol); err != nil {
		e.logger.Error().
			Err(err).
			Str("symbol", symbol).
			Str("exchange", exchangeName).
			Str("side", string(side)).
			Msg("⚠️ timeout: failed to cancel pending leg — check exchange for stuck order")
		return
	}
	e.logger.Info().
		Str("symbol", symbol).
		Str("exchange", exchangeName).
		Str("side", string(side)).
		Msg("✅ timeout: pending limit cancelled")
	e.markOrderCanceled(ctx, orderID)
}

// emergencyCloseLeg market-closes a leg that already filled, after its partner timed out without filling.
// Thin wrapper around submitEmergencyClose with timeout-specific log messages.
func (e *Engine) emergencyCloseLeg(ctx context.Context, client exchange.Provider, pos *Position, side models.OrderSide, exchangeName string) {
	e.submitEmergencyClose(ctx, client, pos, side, exchangeName, "timeout: filled leg emergency market-closed after partner failed to fill")
}

// submitEmergencyClose builds and sends a market reduce-only close order for the given side,
// indexes its orderID in pm.byOrderID so the execution event can route back and update the DB
// row with the actual fill price/qty, and schedules a delayed pm.Delete to clean up later.
//
// Used by both the fill-timeout watcher and partner-failed cleanup paths (for any case where
// a limit leg has actually filled but the arbitrage trade can't proceed and we must flatten
// that side on the exchange).
//
// successMsg is the log message printed on successful CloseOrder; pass a context-specific
// description (e.g. "timeout: ..." vs "partner-failed: ...").
func (e *Engine) submitEmergencyClose(
	ctx context.Context,
	client exchange.Provider,
	pos *Position,
	side models.OrderSide,
	exchangeName string,
	successMsg string,
) {
	vol, err := e.qtyForExchange(pos.QtyCoins, pos.Symbol, exchangeName)
	if err != nil {
		e.logger.Error().Err(err).Str("symbol", pos.Symbol).Str("side", string(side)).Msg("⚠️ emergency close: failed to convert qty — leg may be unhedged on exchange")
		return
	}
	closeOrder, err := e.buildOrder(pos.Symbol, side, vol, exchangeName, nil) // nil price = market
	if err != nil {
		e.logger.Error().Err(err).Str("symbol", pos.Symbol).Str("side", string(side)).Msg("⚠️ emergency close: failed to build order — leg may be unhedged on exchange")
		return
	}
	closeOrder.ID = uuid.New()

	// MarkCleanup is idempotent (sets state = TimedOut). Watcher path reaches TimedOut via
	// OnOpenTimeout; partner-failed path needs this explicit transition.
	pos.MarkCleanup()
	leg := Leg{OrderID: closeOrder.ID}
	pos.SetEmergencyCloseLeg(side, leg)
	if side == models.OrderSideBuy {
		e.pm.IndexCloseLeg(pos, closeOrder.ID, uuid.Nil)
	} else {
		e.pm.IndexCloseLeg(pos, uuid.Nil, closeOrder.ID)
	}

	e.saveOrder(&closeOrder)

	if err := client.CloseOrder(ctx, &closeOrder); err != nil {
		e.logger.Error().
			Err(err).
			Str("symbol", pos.Symbol).
			Str("exchange", exchangeName).
			Str("side", string(side)).
			Msg("⚠️ emergency close FAILED — VERIFY EXCHANGE for open position")
		e.markOrderRejected(ctx, closeOrder.ID)
		// still schedule the delete — otherwise the close-leg orderID stays in pm.byOrderID forever
		e.scheduleDelayedDelete(ctx, pos)
		return
	}
	e.logger.Warn().
		Str("symbol", pos.Symbol).
		Str("exchange", exchangeName).
		Str("side", string(side)).
		Stringer("qty", vol).
		Msg("🛡 " + successMsg)
	// Schedule delete so the close-leg byOrderID entry is cleaned up after the execution event
	// has had a chance to route through and update the DB row. Watcher path also calls this from
	// watchFillTimeout — pm.Delete is idempotent, double-call is harmless.
	e.scheduleDelayedDelete(ctx, pos)
}
