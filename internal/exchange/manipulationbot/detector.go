package manipulationbot

import (
	"fmt"
	"math"
	"time"
)

type signal struct {
	Symbol     string
	Ts         time.Time
	SpotATR    float64
	PerpATR    float64
	SpotATRPct float64
	PerpATRPct float64
	ATRRatio   float64
}

type symbolState struct {
	spot      *marketWindow
	perp      *marketWindow
	lastCheck time.Time
	lastAlert time.Time
}

type detector struct {
	cfg Config
}

func newDetector(cfg Config) *detector {
	return &detector{cfg: cfg}
}

func newSymbolState(cfg Config) *symbolState {
	windowSizeSec := int64(cfg.WindowSize / time.Second)
	if windowSizeSec < 2 {
		windowSizeSec = 2
	}

	return &symbolState{
		spot: newMarketWindow(windowSizeSec),
		perp: newMarketWindow(windowSizeSec),
	}
}

func (d *detector) evaluate(symbol string, state *symbolState, startedAt time.Time) *signal {
	if time.Since(startedAt) < d.cfg.StartupDelay {
		return nil
	}

	if time.Since(state.lastCheck) < d.cfg.CheckInterval {
		return nil
	}
	state.lastCheck = time.Now()

	nowSec := minInt64(state.spot.lastTs, state.perp.lastTs)
	lookbackSec := int64(d.cfg.WindowSize / time.Second)
	spotSnap, ok := state.spot.Snapshot(nowSec, lookbackSec)
	if !ok {
		return nil
	}

	perpSnap, ok := state.perp.Snapshot(nowSec, lookbackSec)
	if !ok {
		return nil
	}

	atrRatio := safeDiv(spotSnap.ATRPct, perpSnap.ATRPct)

	if spotSnap.ATRPct < d.cfg.MinSpotATRPct {
		return nil
	}
	if atrRatio < d.cfg.MinATRRatio {
		return nil
	}
	if time.Since(state.lastAlert) < d.cfg.AlertCooldown {
		return nil
	}

	state.lastAlert = time.Now()

	return &signal{
		Symbol:     symbol,
		Ts:         time.Unix(nowSec, 0).UTC(),
		SpotATR:    spotSnap.ATR,
		PerpATR:    perpSnap.ATR,
		SpotATRPct: spotSnap.ATRPct,
		PerpATRPct: perpSnap.ATRPct,
		ATRRatio:   atrRatio,
	}
}

func (s *signal) Message(exchangeName string) string {
	return fmt.Sprintf(
		"<b>ATR spot-vs-perp signal</b>\n"+
			"Exchange: <b>%s</b>\n"+
			"Symbol: <b>%s</b>\n"+
			"Spot ATR: <b>%.6f</b>\n"+
			"Perp ATR: <b>%.6f</b>\n"+
			"Spot ATR%%: <b>%.3f%%</b>\n"+
			"Perp ATR%%: <b>%.3f%%</b>\n"+
			"ATR ratio spot/perp: <b>%.2f</b>",
		exchangeName,
		s.Symbol,
		s.SpotATR,
		s.PerpATR,
		s.SpotATRPct,
		s.PerpATRPct,
		s.ATRRatio,
	)
}

func minInt64(a int64, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

func safeDiv(num float64, den float64) float64 {
	if den == 0 {
		if num == 0 {
			return 0
		}
		return math.Inf(1)
	}

	return num / den
}
