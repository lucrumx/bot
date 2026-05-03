package arbitragebot

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/lucrumx/bot/internal/exchange"
	"github.com/lucrumx/bot/internal/models"
)

func (a *ArbitrageBot) updateOrderInfoAndCalcSpreadProfit(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case orderID := <-a.engine.chOrdersToUpdate:
			order, err := a.orderRepo.GetByID(ctx, orderID)
			if err != nil {
				a.logger.Error().Err(err).Str("order_id", orderID.String()).Msg("Failed to get order by ID from DB")
				continue
			}

			var client exchange.Provider
			for _, cl := range a.clients {
				if order.ExchangeName == cl.GetExchangeName() {
					client = cl
					break
				}
			}
			if client == nil {
				a.logger.Error().Str("exchange", order.ExchangeName).Msg("updateOrderInfoAndCalcSpreadProfit | No client found for exchange")
				continue
			}

			orderInfo, err := client.GetOrder(ctx, order.ID, order.ExchangeOrderID, order.Symbol)
			if err != nil {
				a.logger.Error().Err(err).Str("order_id", orderID.String()).Str("exchange", order.ExchangeName).Msg("Failed to get order info from exchange")
				continue
			}

			err = a.orderRepo.UpdatePartialy(ctx, orderID, OrderPatch{
				AvgPrice: &orderInfo.AvgPrice,
				Fees:     &orderInfo.Fees,
				Profit:   &orderInfo.Profit,
			})
			if err != nil {
				a.logger.Error().Err(err).Str("order_id", orderID.String()).Msg("Failed to update order info in DB")
				continue
			}

			err = a.fillProfit(ctx, order.ID)
			if err != nil {
				a.logger.Error().Err(err).Str("order_id", orderID.String()).Msg("Failed to calculate spread profit")
				continue
			}
		}
	}
}

func (a *ArbitrageBot) fillProfit(ctx context.Context, orderID uuid.UUID) error {
	spread, err := a.arbitrageSpreadRepo.FindOneByOrderID(ctx, orderID)
	if err != nil {
		return err
	}

	if spread.OpenBuyOrderID != uuid.Nil && spread.OpenSellOrderID != uuid.Nil &&
		spread.CloseBuyOrderID != uuid.Nil && spread.CloseSellOrderID != uuid.Nil {

		openBuyOrder, err := a.orderRepo.GetByID(ctx, spread.OpenBuyOrderID)
		if err != nil {
			return err
		}

		openSellOrder, err := a.orderRepo.GetByID(ctx, spread.OpenSellOrderID)
		if err != nil {
			return err
		}

		closeBuyOrder, err := a.orderRepo.GetByID(ctx, spread.CloseBuyOrderID)
		if err != nil {
			return err
		}

		closeSellOrder, err := a.orderRepo.GetByID(ctx, spread.CloseSellOrderID)
		if err != nil {
			return err
		}

		qty := openBuyOrder.Quantity
		profitA := (closeSellOrder.AvgPrice.Sub(openBuyOrder.AvgPrice)).Mul(qty) // exchange A: buy open → sell close
		profitB := (openSellOrder.AvgPrice.Sub(closeBuyOrder.AvgPrice)).Mul(qty) // exchange B: sell open → buy close

		profit := profitA.Add(profitB).Add(openBuyOrder.Fees).Add(openSellOrder.Fees).Add(closeBuyOrder.Fees).Add(closeSellOrder.Fees)

		err = a.arbitrageSpreadRepo.Update(ctx, &models.ArbitrageSpread{
			Profit:    profit,
			UpdatedAt: time.Now(),
		}, FindFilter{
			ID: spread.ID,
		})
		if err != nil {
			return err
		}
	}

	return nil
}
