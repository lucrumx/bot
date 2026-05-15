package arbitragebot

import (
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/lucrumx/bot/internal/exchange"
	"github.com/lucrumx/bot/internal/models"
)

// coinStep returns the common coin step that satisfies both exchanges' VolStep*ContractSize.
func (e *Engine) coinStep(symbol, exchange1, exchange2 string) (decimal.Decimal, error) {
	inst1, err := e.instrumentFor(symbol, exchange1)
	if err != nil {
		return decimal.Zero, err
	}
	inst2, err := e.instrumentFor(symbol, exchange2)
	if err != nil {
		return decimal.Zero, err
	}
	return lcmDecimal(inst1.VolStep.Mul(inst1.ContractSize), inst2.VolStep.Mul(inst2.ContractSize)), nil
}

func (e *Engine) instrumentFor(symbol, exchangeName string) (exchange.Instrument, error) {
	instruments, ok := e.instruments[exchangeName]
	if !ok {
		return exchange.Instrument{}, fmt.Errorf("no instrument data for exchange %s", exchangeName)
	}
	inst, ok := instruments[symbol]
	if !ok {
		return exchange.Instrument{}, fmt.Errorf("no instrument %s on %s", symbol, exchangeName)
	}
	return inst, nil
}

// qtyForExchange converts coin qty to the exchange-specific vol (qty / contractSize).
func (e *Engine) qtyForExchange(qty decimal.Decimal, symbol, exchangeName string) (decimal.Decimal, error) {
	inst, err := e.instrumentFor(symbol, exchangeName)
	if err != nil {
		return decimal.Zero, err
	}
	return qty.Div(inst.ContractSize), nil
}

// buildOrder constructs a models.Order. price=nil → market order; price!=nil → limit order with
// price aligned to the exchange's PriceStep (so the exchange doesn't reject for an invalid tick).
func (e *Engine) buildOrder(symbol string, side models.OrderSide, qty decimal.Decimal, exchangeName string, price *decimal.Decimal) (models.Order, error) {
	orderType := models.OrderTypeMarket
	dto := exchange.CreateOrderDto{
		Symbol:       symbol,
		Side:         side,
		Type:         orderType,
		Market:       models.OrderMarketLinear,
		Quantity:     qty,
		ExchangeName: exchangeName,
	}
	if price != nil {
		aligned, err := e.alignPriceToInstrument(*price, symbol, exchangeName)
		if err != nil {
			return models.Order{}, fmt.Errorf("align limit price for %s on %s: %w", symbol, exchangeName, err)
		}
		if !aligned.IsPositive() {
			return models.Order{}, fmt.Errorf("aligned limit price is zero for %s on %s", symbol, exchangeName)
		}
		dto.Type = models.OrderTypeLimit
		dto.Price = aligned
		dto.TimeInForce = models.TimeInForceGTC
	}
	return exchange.MakeOrderStruct(dto)
}

// alignPriceToInstrument rounds price to the nearest multiple of the instrument's PriceStep
// using half-up rounding. Returns price unchanged if PriceStep is zero or non-positive.
func (e *Engine) alignPriceToInstrument(price decimal.Decimal, symbol, exchangeName string) (decimal.Decimal, error) {
	inst, err := e.instrumentFor(symbol, exchangeName)
	if err != nil {
		return decimal.Zero, err
	}
	if !inst.PriceStep.IsPositive() {
		return price, nil
	}
	return price.Div(inst.PriceStep).Round(0).Mul(inst.PriceStep), nil
}

// notional returns the trade size in USDT. Hardcoded for now, will come from config.
func (e *Engine) notional() int64 {
	return 10
}

func (e *Engine) isSilentMode() bool {
	return e.cfg.Exchange.ArbitrageBot.SilentMode
}

// gcdDecimal computes the greatest common divisor of two decimals (Euclidean).
func gcdDecimal(a, b decimal.Decimal) decimal.Decimal {
	for !b.IsZero() {
		a, b = b, a.Mod(b)
	}
	return a
}

// lcmDecimal computes the least common multiple of two decimals. Used to find a coin step
// that is a valid multiple of both exchanges' steps.
func lcmDecimal(a, b decimal.Decimal) decimal.Decimal {
	return a.Mul(b).Div(gcdDecimal(a, b))
}
