package arbitragebot

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PositionState represents the current state of an arbitrage position.
type PositionState int

const (
	// PositionStateOpening — open legs submitted, waiting for fills.
	PositionStateOpening PositionState = iota
	// PositionStateOpeningPendingClose — spread closed before both open legs confirmed.
	PositionStateOpeningPendingClose
	// PositionStateOpen — both open legs confirmed, position is active.
	PositionStateOpen
	// PositionStateClosing — close legs submitted, waiting for fills.
	PositionStateClosing
	// PositionStateTimedOut — fill timeout expired, position is being cleaned up.
	PositionStateTimedOut
)

// PositionTransition is the action Engine must take after a state transition.
type PositionTransition int

const (
	TransitionNone          PositionTransition = iota
	TransitionSubmitClose                      // both open legs confirmed, spread already closed → submit close now
	TransitionEmergencyClose                   // one open leg failed → emergency close the other
	TransitionFullyClosed                      // both close legs confirmed → delete position
)

// Position is an active arbitrage position with an explicit state machine.
// All state mutations are protected by the internal mutex.
type Position struct {
	mu sync.Mutex

	Symbol      string
	BuyExchange string
	SellExchange string
	QtyCoins    decimal.Decimal // qty in coins (base currency), used for close leg sizing

	OpenBuyLeg  Leg
	OpenSellLeg Leg

	CloseBuyLeg  Leg
	CloseSellLeg Leg

	State PositionState
}

// Key returns a unique string key for the position.
func (p *Position) Key() string {
	return positionKey(p.Symbol, p.BuyExchange, p.SellExchange)
}

// OnOpenLegFilled processes a fill event for an open leg.
// Returns the transition Engine should apply.
func (p *Position) OnOpenLegFilled(orderID uuid.UUID, execPrice, execQty decimal.Decimal) (PositionTransition, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.State != PositionStateOpening && p.State != PositionStateOpeningPendingClose {
		return TransitionNone, fmt.Errorf("position %s: unexpected OnOpenLegFilled in state %d", p.Symbol, p.State)
	}

	switch orderID {
	case p.OpenBuyLeg.OrderID:
		p.OpenBuyLeg.Confirmed = true
		p.OpenBuyLeg.AvgPrice = execPrice
		p.OpenBuyLeg.FilledQty = execQty
	case p.OpenSellLeg.OrderID:
		p.OpenSellLeg.Confirmed = true
		p.OpenSellLeg.AvgPrice = execPrice
		p.OpenSellLeg.FilledQty = execQty
	default:
		return TransitionNone, fmt.Errorf("position %s: unknown order ID %s in OnOpenLegFilled", p.Symbol, orderID)
	}

	if !p.bothOpenConfirmed() {
		return TransitionNone, nil
	}

	if p.State == PositionStateOpeningPendingClose {
		p.State = PositionStateClosing
		return TransitionSubmitClose, nil
	}

	p.State = PositionStateOpen
	return TransitionNone, nil
}

// OnCloseLegFilled processes a fill event for a close leg.
// Returns the transition Engine should apply.
func (p *Position) OnCloseLegFilled(orderID uuid.UUID, execPrice, execQty decimal.Decimal) (PositionTransition, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.State != PositionStateClosing {
		return TransitionNone, fmt.Errorf("position %s: unexpected OnCloseLegFilled in state %d", p.Symbol, p.State)
	}

	switch orderID {
	case p.CloseBuyLeg.OrderID:
		p.CloseBuyLeg.Confirmed = true
		p.CloseBuyLeg.AvgPrice = execPrice
		p.CloseBuyLeg.FilledQty = execQty
	case p.CloseSellLeg.OrderID:
		p.CloseSellLeg.Confirmed = true
		p.CloseSellLeg.AvgPrice = execPrice
		p.CloseSellLeg.FilledQty = execQty
	default:
		return TransitionNone, fmt.Errorf("position %s: unknown order ID %s in OnCloseLegFilled", p.Symbol, orderID)
	}

	if !p.bothCloseConfirmed() {
		return TransitionNone, nil
	}

	return TransitionFullyClosed, nil
}

// RequestClose signals that the spread has closed and the position should be closed.
// Returns the transition Engine should apply.
func (p *Position) RequestClose() PositionTransition {
	p.mu.Lock()
	defer p.mu.Unlock()

	switch p.State {
	case PositionStateOpening:
		p.State = PositionStateOpeningPendingClose
		return TransitionNone
	case PositionStateOpen:
		p.State = PositionStateClosing
		return TransitionSubmitClose
	default:
		return TransitionNone
	}
}

// SetOpenLegExchangeOrderID stores the exchange-assigned order ID on the open leg.
// Called after CreateOrder returns to enable cancel by exchange ID (e.g. MEXC).
func (p *Position) SetOpenLegExchangeOrderID(orderID uuid.UUID, exchangeOrderID string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	switch orderID {
	case p.OpenBuyLeg.OrderID:
		p.OpenBuyLeg.ExchangeOrderID = exchangeOrderID
	case p.OpenSellLeg.OrderID:
		p.OpenSellLeg.ExchangeOrderID = exchangeOrderID
	}
}

// SetCloseLegIDs registers the close order IDs so execution events can be matched.
// Must be called after state is already PositionStateClosing.
func (p *Position) SetCloseLegIDs(buyLeg, sellLeg Leg) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.CloseBuyLeg = buyLeg
	p.CloseSellLeg = sellLeg
}

// GetState returns the current state under the mutex.
func (p *Position) GetState() PositionState {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.State
}

func (p *Position) bothOpenConfirmed() bool {
	return p.OpenBuyLeg.Confirmed && p.OpenSellLeg.Confirmed
}

func (p *Position) bothCloseConfirmed() bool {
	return p.CloseBuyLeg.Confirmed && p.CloseSellLeg.Confirmed
}

// OpenTimeoutInfo describes the state of open legs at the moment the fill timeout fired.
type OpenTimeoutInfo struct {
	BuyFilled   bool
	SellFilled  bool
	BuyOrderID  uuid.UUID
	SellOrderID uuid.UUID
	// ExchangeOrderIDs needed for exchanges that cancel by their own ID (e.g. MEXC)
	BuyExchangeOrderID  string
	SellExchangeOrderID string
}

// OnOpenTimeout is called when the fill timeout expires for open legs.
// Returns (false, zero) if the position already transitioned — no cleanup needed.
// Returns (true, info) if position was still Opening and needs cancel/emergency close.
func (p *Position) OnOpenTimeout() (bool, OpenTimeoutInfo) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.State != PositionStateOpening && p.State != PositionStateOpeningPendingClose {
		return false, OpenTimeoutInfo{}
	}

	p.State = PositionStateTimedOut

	return true, OpenTimeoutInfo{
		BuyFilled:           p.OpenBuyLeg.Confirmed,
		SellFilled:          p.OpenSellLeg.Confirmed,
		BuyOrderID:          p.OpenBuyLeg.OrderID,
		SellOrderID:         p.OpenSellLeg.OrderID,
		BuyExchangeOrderID:  p.OpenBuyLeg.ExchangeOrderID,
		SellExchangeOrderID: p.OpenSellLeg.ExchangeOrderID,
	}
}

func positionKey(symbol, buyExchange, sellExchange string) string {
	return symbol + "#" + buyExchange + "#" + sellExchange
}
