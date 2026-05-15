package arbitragebot

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/lucrumx/bot/internal/exchange"
	"github.com/lucrumx/bot/internal/models"
)

// saveSpread inserts a row in the arbitrage_spreads table for the given event. pos may be nil when
// running in silent mode (no real position).
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

// saveCloseOrderIDsToSpread back-fills the close order IDs onto the existing spread row.
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

// markSpreadFailed flips the spread row to ArbitrageSpreadFailed.
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

// saveOrder persists an order row before CreateOrder/CloseOrder so any subsequent execution event
// is guaranteed to find a row to update.
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

func (e *Engine) markOrderCanceled(ctx context.Context, orderID uuid.UUID) {
	if e.orderRepo == nil {
		return
	}
	canceled := models.OrderStatusCanceled
	if err := e.orderRepo.UpdatePartialy(ctx, orderID, OrderPatch{Status: &canceled}); err != nil {
		e.logger.Error().Err(err).Str("order_id", orderID.String()).Msg("execution: failed to mark order canceled")
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
