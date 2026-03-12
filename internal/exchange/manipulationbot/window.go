package manipulationbot

import (
	"math"

	"github.com/lucrumx/bot/internal/exchange"
)

type marketWindow struct {
	sizeSec    int64
	highs      []float64
	lows       []float64
	closes     []float64
	timestamps []int64
	lastPrice  float64
	lastTs     int64
}

type marketSnapshot struct {
	ATR    float64
	ATRPct float64
}

func newMarketWindow(sizeSec int64) *marketWindow {
	sizeSec += 5

	return &marketWindow{
		sizeSec:    sizeSec,
		highs:      make([]float64, sizeSec),
		lows:       make([]float64, sizeSec),
		closes:     make([]float64, sizeSec),
		timestamps: make([]int64, sizeSec),
	}
}

func (w *marketWindow) AddTrade(trade exchange.Trade) {
	tsSec := trade.Ts / 1000
	if tsSec <= 0 {
		return
	}

	if w.lastTs == 0 {
		w.setSecond(tsSec, trade.Price)
		w.lastTs = tsSec
		w.lastPrice = trade.Price
		return
	}

	if tsSec > w.lastTs {
		w.fillGaps(tsSec)
		w.setSecond(tsSec, trade.Price)
	} else {
		idx := int(tsSec % w.sizeSec)
		if w.timestamps[idx] != tsSec {
			w.setSecond(tsSec, trade.Price)
		} else {
			if trade.Price > w.highs[idx] {
				w.highs[idx] = trade.Price
			}
			if w.lows[idx] == 0 || trade.Price < w.lows[idx] {
				w.lows[idx] = trade.Price
			}
			w.closes[idx] = trade.Price
		}
	}

	w.lastTs = maxInt64(w.lastTs, tsSec)
	w.lastPrice = trade.Price
}

func (w *marketWindow) fillGaps(targetTs int64) {
	start := w.lastTs + 1
	if targetTs-start > w.sizeSec {
		start = targetTs - w.sizeSec
	}

	for ts := start; ts < targetTs; ts++ {
		w.setSecond(ts, w.lastPrice)
	}
}

func (w *marketWindow) setSecond(ts int64, price float64) {
	idx := int(ts % w.sizeSec)
	w.highs[idx] = price
	w.lows[idx] = price
	w.closes[idx] = price
	w.timestamps[idx] = ts
}

func (w *marketWindow) Snapshot(nowSec int64, lookbackSec int64) (marketSnapshot, bool) {
	if w.lastTs == 0 || lookbackSec <= 1 {
		return marketSnapshot{}, false
	}

	if nowSec > w.lastTs {
		nowSec = w.lastTs
	}

	startSec := nowSec - lookbackSec
	if startSec <= 0 || nowSec-startSec >= w.sizeSec {
		return marketSnapshot{}, false
	}

	prevClose, ok := w.closeAt(startSec)
	if !ok || prevClose <= 0 {
		return marketSnapshot{}, false
	}

	var trSum float64
	var bars int64

	for ts := startSec + 1; ts <= nowSec; ts++ {
		idx := int(ts % w.sizeSec)
		if w.timestamps[idx] != ts {
			return marketSnapshot{}, false
		}

		high := w.highs[idx]
		low := w.lows[idx]
		klose := w.closes[idx]
		if high <= 0 || low <= 0 || klose <= 0 || prevClose <= 0 {
			return marketSnapshot{}, false
		}

		tr := math.Max(high-low, math.Max(math.Abs(high-prevClose), math.Abs(low-prevClose)))
		trSum += tr
		bars++
		prevClose = klose
	}

	if bars == 0 || prevClose <= 0 {
		return marketSnapshot{}, false
	}

	atr := trSum / float64(bars)

	return marketSnapshot{
		ATR:    atr,
		ATRPct: atr / prevClose * 100,
	}, true
}

func (w *marketWindow) closeAt(ts int64) (float64, bool) {
	idx := int(ts % w.sizeSec)
	if w.timestamps[idx] != ts {
		return 0, false
	}

	return w.closes[idx], true
}

func maxInt64(a int64, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
