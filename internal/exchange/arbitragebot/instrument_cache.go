package arbitragebot

import (
	"context"
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/lucrumx/bot/internal/exchange"
)

// instrumentCache stores contract specifications per exchange, loaded once on startup.
type instrumentCache struct {
	// data[exchangeName][symbol] = Instrument
	data map[string]map[string]exchange.Instrument
}

func newInstrumentCache() *instrumentCache {
	return &instrumentCache{
		data: make(map[string]map[string]exchange.Instrument),
	}
}

// Load fetches instruments from all clients and populates the cache.
func (c *instrumentCache) Load(ctx context.Context, clients []exchange.Provider) error {
	for _, client := range clients {
		instruments, err := client.GetInstruments(ctx)
		if err != nil {
			return fmt.Errorf("instrument cache: failed to load from %s: %w", client.GetExchangeName(), err)
		}
		c.data[client.GetExchangeName()] = instruments
	}
	return nil
}

// VolStep returns the maximum vol step across two exchanges for the given symbol.
// This is the step size that both exchanges can satisfy simultaneously.
func (c *instrumentCache) VolStep(symbol, exchange1, exchange2 string) (decimal.Decimal, error) {
	step1, err := c.volStepFor(symbol, exchange1)
	if err != nil {
		return decimal.Zero, err
	}
	step2, err := c.volStepFor(symbol, exchange2)
	if err != nil {
		return decimal.Zero, err
	}
	return decimal.Max(step1, step2), nil
}

func (c *instrumentCache) volStepFor(symbol, exchangeName string) (decimal.Decimal, error) {
	instruments, ok := c.data[exchangeName]
	if !ok {
		return decimal.Zero, fmt.Errorf("instrument cache: no data for exchange %s", exchangeName)
	}
	instrument, ok := instruments[symbol]
	if !ok {
		return decimal.Zero, fmt.Errorf("instrument cache: no instrument %s on %s", symbol, exchangeName)
	}
	return instrument.VolStep, nil
}
