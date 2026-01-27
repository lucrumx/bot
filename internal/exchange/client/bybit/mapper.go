package bybit

import (
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/lucrumx/bot/internal/exchange"
)

// mapTicker converts Bybit TickerDTO to exchange.Ticker
func mapTicker(d TickerDTO) (exchange.Ticker, error) {
	var t exchange.Ticker
	var err error

	t.Symbol = d.Symbol

	if t.LastPrice, err = parseDecimal(d.LastPrice); err != nil {
		return t, fmt.Errorf("LastPrice: %w", err)
	}
	if t.IndexPrice, err = parseDecimal(d.IndexPrice); err != nil {
		return t, fmt.Errorf("indexPrice: %w", err)
	}
	if t.MarkPrice, err = parseDecimal(d.MarkPrice); err != nil {
		return t, fmt.Errorf("markPrice: %w", err)
	}
	if t.PrevPrice24h, err = parseDecimal(d.PrevPrice24h); err != nil {
		return t, fmt.Errorf("prevPrice24h: %w", err)
	}
	if t.Price24hPcnt, err = parseDecimal(d.Price24hPcnt); err != nil {
		return t, fmt.Errorf("price24hPcnt: %w", err)
	}
	if t.HighPrice24h, err = parseDecimal(d.HighPrice24h); err != nil {
		return t, fmt.Errorf("highPrice24h: %w", err)
	}
	if t.LowPrice24h, err = parseDecimal(d.LowPrice24h); err != nil {
		return t, fmt.Errorf("lowPrice24h: %w", err)
	}
	if t.PrevPrice1h, err = parseDecimal(d.PrevPrice1h); err != nil {
		return t, fmt.Errorf("prevPrice1h: %w", err)
	}
	if t.OpenInterest, err = parseDecimal(d.OpenInterest); err != nil {
		return t, fmt.Errorf("openInterest: %w", err)
	}
	if t.OpenInterestValue, err = parseDecimal(d.OpenInterestValue); err != nil {
		return t, fmt.Errorf("OpenInterestValue: %w", err)
	}
	if t.Turnover24h, err = parseDecimal(d.Turnover24h); err != nil {
		return t, fmt.Errorf("turnover24h: %w", err)
	}

	return t, nil
}

// mapWsTrade maps a WsTradeDTO object to an exchange.Trade object and calculates the USDT amount
// uses while read and parse a websocket trade message.
func mapWsTrade(d wsTradeDTO) (exchange.Trade, error) {
	var trade exchange.Trade
	var err error

	trade.Symbol = d.Symbol
	trade.Ts = d.T
	trade.Side = d.Side

	if trade.Price, err = parseDecimal(d.Price); err != nil {
		return trade, fmt.Errorf("price: %w", err)
	}

	if trade.Volume, err = parseDecimal(d.Volume); err != nil {
		return trade, fmt.Errorf("volume: %w", err)
	}

	trade.USDTAmount = trade.Price.Mul(trade.Volume)

	return trade, nil
}

// parseDecimal безопасно парсит строку в decimal, возвращая 0 для пустых строк
func parseDecimal(s string) (decimal.Decimal, error) {
	if s == "" {
		return decimal.Zero, nil
	}
	return decimal.NewFromString(s)
}
