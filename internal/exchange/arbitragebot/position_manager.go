package arbitragebot

import (
	"sync"

	"github.com/google/uuid"
)

// PositionManager manages the collection of active positions and the runtime blacklist.
// All methods are safe for concurrent use.
type PositionManager struct {
	mu        sync.Mutex
	positions map[string]*Position      // key → position
	byOrderID map[uuid.UUID]*Position   // orderID → position (for fast lookup)
	blacklist map[string]struct{}       // symbol → blacklisted
}

func newPositionManager() *PositionManager {
	return &PositionManager{
		positions: make(map[string]*Position),
		byOrderID: make(map[uuid.UUID]*Position),
		blacklist:  make(map[string]struct{}),
	}
}

// Add registers a new position and indexes its open leg order IDs.
func (m *PositionManager) Add(pos *Position) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.positions[pos.Key()] = pos
	m.byOrderID[pos.OpenBuyLeg.OrderID] = pos
	m.byOrderID[pos.OpenSellLeg.OrderID] = pos
}

// IndexCloseLeg adds close leg order IDs to the lookup index.
func (m *PositionManager) IndexCloseLeg(pos *Position, buyOrderID, sellOrderID uuid.UUID) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if buyOrderID != uuid.Nil {
		m.byOrderID[buyOrderID] = pos
	}
	if sellOrderID != uuid.Nil {
		m.byOrderID[sellOrderID] = pos
	}
}

// Delete removes the position and all its order ID index entries.
func (m *PositionManager) Delete(pos *Position) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.positions, pos.Key())
	delete(m.byOrderID, pos.OpenBuyLeg.OrderID)
	delete(m.byOrderID, pos.OpenSellLeg.OrderID)
	delete(m.byOrderID, pos.CloseBuyLeg.OrderID)
	delete(m.byOrderID, pos.CloseSellLeg.OrderID)
}

// FindByOrderID returns the position associated with the given order ID, or nil.
func (m *PositionManager) FindByOrderID(orderID uuid.UUID) *Position {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.byOrderID[orderID]
}

// FindByKey returns the position for the given symbol+exchanges key, or nil.
func (m *PositionManager) FindByKey(symbol, buyExchange, sellExchange string) *Position {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.positions[positionKey(symbol, buyExchange, sellExchange)]
}

// HasOverlap returns true if any active position uses the same symbol
// and shares at least one exchange with the given pair.
func (m *PositionManager) HasOverlap(symbol, buyExchange, sellExchange string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, pos := range m.positions {
		if pos.Symbol != symbol {
			continue
		}
		if pos.BuyExchange == buyExchange || pos.BuyExchange == sellExchange ||
			pos.SellExchange == buyExchange || pos.SellExchange == sellExchange {
			return true
		}
	}
	return false
}

// Count returns the number of active positions.
func (m *PositionManager) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.positions)
}

// Blacklist adds the symbol to the blacklist and removes its position.
func (m *PositionManager) Blacklist(pos *Position) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.blacklist[pos.Symbol] = struct{}{}
	delete(m.positions, pos.Key())
	delete(m.byOrderID, pos.OpenBuyLeg.OrderID)
	delete(m.byOrderID, pos.OpenSellLeg.OrderID)
}

// IsBlacklisted returns true if the symbol is blacklisted.
func (m *PositionManager) IsBlacklisted(symbol string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.blacklist[symbol]
	return ok
}
