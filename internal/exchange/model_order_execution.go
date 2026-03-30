package exchange

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type OrderExecutionEvent struct {
	OrderID    uuid.UUID
	ExecPrice  decimal.Decimal
	ExecQty    decimal.Decimal
	ExecValue  decimal.Decimal
	LeavesQty  decimal.Decimal
	OrderPrice decimal.Decimal
	OrderQty   decimal.Decimal
}

// type ExecutedOrders map[uuid.UUID]OrderExecutionEvent
