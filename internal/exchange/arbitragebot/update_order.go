package arbitragebot

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/lucrumx/bot/internal/exchange"
	"github.com/lucrumx/bot/internal/models"
)

const (
	profitCalcInterval = time.Minute
	advisoryLockID     = 777_001 // уникальный ID для pg_advisory_lock
)

// updateOrderInfoAndCalcSpreadProfit периодически ищет закрытые спреды без посчитанного профита,
// обновляет информацию по ордерам с бирж и считает профит.
func (a *ArbitrageBot) updateOrderInfoAndCalcSpreadProfit(ctx context.Context) {
	ticker := time.NewTicker(profitCalcInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.processUnfilledSpreads(ctx)
		}
	}
}

func (a *ArbitrageBot) processUnfilledSpreads(ctx context.Context) {
	// pg_advisory_xact_lock — лок привязан к транзакции, снимается при commit/rollback,
	// гарантирует одно соединение и что только один инстанс бота считает профит одновременно
	tx := a.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		a.logger.Error().Err(tx.Error).Msg("profit-calc: failed to begin transaction")
		return
	}
	defer tx.Rollback()

	if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", advisoryLockID).Error; err != nil {
		a.logger.Error().Err(err).Msg("profit-calc: failed to acquire advisory lock")
		return
	}

	// Спреды со статусом CLOSED, у которых заполнены все 4 order ID, но profit ещё не посчитан
	spreads, err := a.arbitrageSpreadRepo.FindAll(ctx, FindFilter{
		Status: []models.ArbitrageSpreadStatus{models.ArbitrageSpreadClosed},
	})
	if err != nil {
		a.logger.Error().Err(err).Msg("profit-calc: failed to find closed spreads")
		return
	}

	for _, spread := range spreads {
		if spread.OpenBuyOrderID == uuid.Nil || spread.OpenSellOrderID == uuid.Nil ||
			spread.CloseBuyOrderID == uuid.Nil || spread.CloseSellOrderID == uuid.Nil {
			continue
		}
		if spread.Profit != nil {
			continue
		}

		if err := a.processSpread(ctx, spread); err != nil {
			a.logger.Error().Err(err).Str("spread_id", spread.ID.String()).Msg("profit-calc: failed to process spread")
		}
	}

	tx.Commit()
}

func (a *ArbitrageBot) processSpread(ctx context.Context, spread *models.ArbitrageSpread) error {
	orderIDs := []uuid.UUID{
		spread.OpenBuyOrderID, spread.OpenSellOrderID,
		spread.CloseBuyOrderID, spread.CloseSellOrderID,
	}

	orders := make(map[uuid.UUID]*models.Order, 4)
	for _, id := range orderIDs {
		order, err := a.orderRepo.GetByID(ctx, id)
		if err != nil {
			return err
		}

		// Если avg_price ещё не заполнен — подтягиваем с биржи
		if order.AvgPrice.IsZero() {
			if err := a.enrichOrderFromExchange(ctx, order); err != nil {
				return err
			}
		}

		orders[id] = order
	}

	openBuy := orders[spread.OpenBuyOrderID]
	openSell := orders[spread.OpenSellOrderID]
	closeBuy := orders[spread.CloseBuyOrderID]
	closeSell := orders[spread.CloseSellOrderID]

	qty := openBuy.Quantity
	profitA := closeSell.AvgPrice.Sub(openBuy.AvgPrice).Mul(qty) // exchange A: buy open → sell close
	profitB := openSell.AvgPrice.Sub(closeBuy.AvgPrice).Mul(qty) // exchange B: sell open → buy close
	fees := openBuy.Fees.Add(openSell.Fees).Add(closeBuy.Fees).Add(closeSell.Fees)
	profit := profitA.Add(profitB).Add(fees)

	a.logger.Info().
		Str("spread_id", spread.ID.String()).
		Str("symbol", spread.Symbol).
		Stringer("profit", profit).
		Stringer("fees", fees).
		Msg("profit-calc: spread profit calculated")

	return a.arbitrageSpreadRepo.Update(ctx, &models.ArbitrageSpread{
		Profit:    &profit,
		UpdatedAt: time.Now(),
	}, FindFilter{
		ID: spread.ID,
	})
}

func (a *ArbitrageBot) enrichOrderFromExchange(ctx context.Context, order *models.Order) error {
	var client exchange.Provider
	for _, cl := range a.clients {
		if order.ExchangeName == cl.GetExchangeName() {
			client = cl
			break
		}
	}
	if client == nil {
		return fmt.Errorf("no client found for exchange %s", order.ExchangeName)
	}

	orderInfo, err := client.GetOrder(ctx, order.ID, order.ExchangeOrderID, order.Symbol)
	if err != nil {
		return err
	}

	filledStatus := models.OrderStatusFilled
	patch := OrderPatch{
		AvgPrice: &orderInfo.AvgPrice,
		Fees:     &orderInfo.Fees,
		Status:   &filledStatus,
	}
	if !orderInfo.Profit.IsZero() {
		patch.Profit = &orderInfo.Profit
	}

	if err := a.orderRepo.UpdatePartialy(ctx, order.ID, patch); err != nil {
		return err
	}

	// обновляем in-memory для расчёта профита в этом же цикле
	order.AvgPrice = orderInfo.AvgPrice
	order.Fees = orderInfo.Fees
	order.Status = models.OrderStatusFilled

	return nil
}
