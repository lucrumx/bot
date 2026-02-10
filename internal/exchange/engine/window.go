package engine

import (
	"time"

	"github.com/lucrumx/bot/internal/exchange"
)

// Window tracks price data over a fixed time window, enabling analysis of trends and alerts for significant changes.
type Window struct {
	prices     []float64
	timestamps []int64
	windowSize int64

	lastCheck      time.Time
	lastAlertTime  time.Time
	lastAlertLevel float64

	lastPrice float64
	lastTs    int64
}

// NewWindow creates a new Window with the specified size.
func NewWindow(size int) *Window {
	// запас, чтобы interval гарантированно помещался
	size = size + 50

	return &Window{
		windowSize: int64(size),
		prices:     make([]float64, size),
		timestamps: make([]int64, size),
	}
}

// fillGaps fills missing timestamps in the window with the last known price to maintain continuity in the data series.
// fill only during real trades
func (w *Window) fillGaps(targetTs int64) {
	if w.lastTs == 0 || targetTs <= w.lastTs {
		return
	}

	start := w.lastTs + 1
	if targetTs-start > w.windowSize {
		start = targetTs - w.windowSize
	}

	for t := start; t < targetTs; t++ {
		idx := int(t % w.windowSize)
		w.prices[idx] = w.lastPrice
		w.timestamps[idx] = t
	}
}

// AddTrade integrates a new trade into the window, updating prices, timestamps, and filling any gaps in the time series.
func (w *Window) AddTrade(trade exchange.Trade) {
	ts := trade.Ts / 1000

	// первый трейд — инициализация
	if w.lastTs == 0 {
		idx := int(ts % w.windowSize)
		w.prices[idx] = trade.Price
		w.timestamps[idx] = ts
		w.lastPrice = trade.Price
		w.lastTs = ts
		return
	}

	w.fillGaps(ts)

	idx := int(ts % w.windowSize)
	w.prices[idx] = trade.Price
	w.timestamps[idx] = ts

	w.lastPrice = trade.Price
	w.lastTs = ts
}

// CheckGrow evaluates if the price increase over a given interval exceeds a target percentage.
// It returns the percentage change and a boolean indicating whether the growth condition is met or not.
func (w *Window) CheckGrow(interval int, targetPercent float64) (float64, bool) {
	if int64(interval) >= w.windowSize || w.lastTs == 0 {
		return 0, false
	}

	now := time.Now().Unix()
	pastTs := now - int64(interval)

	// ---- текущая цена ----
	var currPrice float64
	var currTs int64

	if now <= w.lastTs {
		idx := int(now % w.windowSize)
		if w.timestamps[idx] != now {
			return 0, false
		}
		currPrice = w.prices[idx]
		currTs = now
	} else {
		// новых трейдов не было
		currPrice = w.lastPrice
		currTs = w.lastTs
	}

	// ---- прошлая цена ----
	if pastTs < currTs-w.windowSize {
		return 0, false // вышли за окно
	}

	pastIdx := int(pastTs % w.windowSize)
	if w.timestamps[pastIdx] != pastTs {
		return 0, false
	}

	pastPrice := w.prices[pastIdx]
	if pastPrice == 0 {
		return 0, false
	}

	change := (currPrice - pastPrice) / pastPrice * 100

	if change > targetPercent {
		return change, true
	}
	return 0, false
}

// CanCheck determines if the specified minimum interval has elapsed since the last check and updates the last check time.
func (w *Window) CanCheck(minInterval time.Duration) bool {
	if time.Since(w.lastCheck) >= minInterval {
		w.lastCheck = time.Now()
		return true
	}
	return false
}

// GetAlertState returns the last alert time and level, providing the state of the most recent alert.
func (w *Window) GetAlertState() (time.Time, float64) {
	return w.lastAlertTime, w.lastAlertLevel
}

// UpdateAlertState updates the last alert time and level with the specified level.
func (w *Window) UpdateAlertState(level float64) {
	w.lastAlertTime = time.Now()
	w.lastAlertLevel = level
}
