package arbitragebot

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PositionState represents the current state of an arbitrage position.
type PositionState int

const (
	// PositionStateOpening means both orders have been submitted but not yet confirmed.
	PositionStateOpening PositionState = iota
	// PositionStateOpeningPendingClose means spread closed while orders are still being confirmed.
	PositionStateOpeningPendingClose
	// PositionStateOpen means both orders are confirmed and the position is active.
	PositionStateOpen
	// PositionStateClosing means close orders have been submitted.
	PositionStateClosing
)

// OpenPosition represents an active arbitrage position across two exchanges.
type OpenPosition struct {
	Symbol       string
	BuyExchange  string
	SellExchange string
	Qty          decimal.Decimal // quantity in coins (base currency)

	BuyOrderID  uuid.UUID
	SellOrderID uuid.UUID

	BuyConfirmed  bool
	SellConfirmed bool

	CloseBuyOrderID  uuid.UUID
	CloseSellOrderID uuid.UUID

	CloseBuyConfirmed  bool
	CloseSellConfirmed bool

	State PositionState
}

func (p *OpenPosition) bothOpenConfirmed() bool {
	return p.BuyConfirmed && p.SellConfirmed
}

func (p *OpenPosition) bothCloseConfirmed() bool {
	return p.CloseBuyConfirmed && p.CloseSellConfirmed
}
