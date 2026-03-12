package manipulationbot

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/lucrumx/bot/internal/exchange"
)

func TestDetector_EvaluateSignalsOnSpotLedMove(t *testing.T) {
	cfg := DefaultConfig()
	cfg.WindowSize = 10 * time.Second
	cfg.CheckInterval = 0
	cfg.StartupDelay = 0
	cfg.AlertCooldown = time.Hour
	cfg.MinSpotATRPct = 0.10
	cfg.MinATRRatio = 1.5

	d := newDetector(cfg)
	state := newSymbolState(cfg)

	baseTs := time.Now().Add(-time.Minute).UnixMilli()
	for i := 0; i < 12; i++ {
		addSyntheticTrade(state.spot, exchange.CategorySpot, baseTs+int64(i*1000), 100+float64(i)*0.40, 100)
		addSyntheticTrade(state.perp, exchange.CategoryLinear, baseTs+int64(i*1000), 100+float64(i)*0.08, 100)
	}

	sig := d.evaluate("ROAMUSDT", state, time.Now().Add(-time.Hour))
	require.NotNil(t, sig)
	require.Equal(t, "ROAMUSDT", sig.Symbol)
	require.Greater(t, sig.SpotATRPct, cfg.MinSpotATRPct)
	require.Greater(t, sig.ATRRatio, cfg.MinATRRatio)
}

func TestDetector_EvaluateSkipsWhenATRRatioTooLow(t *testing.T) {
	cfg := DefaultConfig()
	cfg.WindowSize = 10 * time.Second
	cfg.CheckInterval = 0
	cfg.StartupDelay = 0
	cfg.AlertCooldown = time.Hour
	cfg.MinSpotATRPct = 0.05
	cfg.MinATRRatio = 1.5

	d := newDetector(cfg)
	state := newSymbolState(cfg)

	baseTs := time.Now().Add(-time.Minute).UnixMilli()
	for i := 0; i < 12; i++ {
		addSyntheticTrade(state.spot, exchange.CategorySpot, baseTs+int64(i*1000), 100+float64(i)*0.10, 100)
		addSyntheticTrade(state.perp, exchange.CategoryLinear, baseTs+int64(i*1000), 100+float64(i)*0.09, 100)
	}

	sig := d.evaluate("ROAMUSDT", state, time.Now().Add(-time.Hour))
	require.Nil(t, sig)
}

func addSyntheticTrade(w *marketWindow, category exchange.Category, tsMs int64, price float64, turnover float64) {
	w.AddTrade(exchange.Trade{
		Symbol:   "ROAMUSDT",
		Category: category,
		Ts:       tsMs,
		Price:    price,
		Volume:   turnover / price,
		Side:     exchange.Buy,
	})
}
