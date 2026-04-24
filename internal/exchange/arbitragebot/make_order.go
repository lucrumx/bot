package arbitragebot

import (
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/lucrumx/bot/internal/exchange"
	"github.com/lucrumx/bot/internal/models"
)

// MakeOrder creates an order struct based on the provided parameters.
func MakeOrder(side models.OrderSide, orderType models.OrderType, market models.OrderMarket, notional int64, ticker *exchange.Ticker, provider exchange.Provider) (models.Order, error) {
	if orderType != models.OrderTypeMarket {
		return models.Order{}, fmt.Errorf("unsupported order type: %s", orderType)
	}

	qty := decimal.NewFromInt(notional).Div(ticker.LastPrice).Round(1)

	return exchange.MakeOrderStruct(exchange.CreateOrderDto{
		Market:       market,
		Symbol:       ticker.Symbol,
		Side:         side,
		Type:         orderType,
		Quantity:     qty,
		ExchangeName: provider.GetExchangeName(),
	})
}

// AlignOrderQty adjusts the order quantity to align with the maximum of the two provided step sizes.
func AlignOrderQty(qty, step1, step2 decimal.Decimal) decimal.Decimal {
	step := decimal.Max(step1, step2)
	return qty.Div(step).Floor().Mul(step)
}
