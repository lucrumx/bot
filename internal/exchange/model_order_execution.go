package exchange

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// OrderExecutionEvent represents the execution of an order.
type OrderExecutionEvent struct {
	OrderID         uuid.UUID
	ExchangeOrderID string
	ExecPrice       decimal.Decimal
	ExecQty         decimal.Decimal
	ExecValue       decimal.Decimal
	LeavesQty       decimal.Decimal
	OrderPrice      decimal.Decimal
	OrderQty        decimal.Decimal
}

// type ExecutedOrders map[uuid.UUID]OrderExecutionEvent
