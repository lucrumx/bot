package exchange

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/lucrumx/bot/internal/models"
)

// CreateOrderDto represent struct for new order. Common for all client
type CreateOrderDto struct {
	OrderID      uuid.UUID
	Market       models.OrderMarket
	Symbol       string
	Side         models.OrderSide
	Type         models.OrderType
	Quantity     decimal.Decimal
	ExchangeName string
	Price        decimal.Decimal // Only for limit orders
}

// MakeOrderStruct construnct models.Order
func MakeOrderStruct(order CreateOrderDto) (models.Order, error) {
	if order.OrderID == uuid.Nil {
		order.OrderID, _ = uuid.NewV7()
	}

	if order.Type == models.OrderTypeLimit && order.Price.LessThanOrEqual(decimal.Zero) {
		return models.Order{}, fmt.Errorf("order price must be greater than 0 for limit order type")
	}

	return models.Order{
		ID:           order.OrderID,
		ExchangeName: order.ExchangeName,
		Status:       models.OrderStatusNew,
		Symbol:       order.Symbol,
		Market:       order.Market,
		Side:         order.Side,
		Type:         order.Type,
		Price:        order.Price,
		Quantity:     order.Quantity,
	}, nil
}
