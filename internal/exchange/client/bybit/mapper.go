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
		return t, fmt.Errorf("IndexPrice: %w", err)
	}
	if t.MarkPrice, err = parseDecimal(d.MarkPrice); err != nil {
		return t, fmt.Errorf("MarkPrice: %w", err)
	}
	if t.PrevPrice24h, err = parseDecimal(d.PrevPrice24h); err != nil {
		return t, fmt.Errorf("PrevPrice24h: %w", err)
	}
	if t.Price24hPcnt, err = parseDecimal(d.Price24hPcnt); err != nil {
		return t, fmt.Errorf("Price24hPcnt: %w", err)
	}
	if t.HighPrice24h, err = parseDecimal(d.HighPrice24h); err != nil {
		return t, fmt.Errorf("HighPrice24h: %w", err)
	}
	if t.LowPrice24h, err = parseDecimal(d.LowPrice24h); err != nil {
		return t, fmt.Errorf("LowPrice24h: %w", err)
	}
	if t.PrevPrice1h, err = parseDecimal(d.PrevPrice1h); err != nil {
		return t, fmt.Errorf("PrevPrice1h: %w", err)
	}
	if t.OpenInterest, err = parseDecimal(d.OpenInterest); err != nil {
		return t, fmt.Errorf("OpenInterest: %w", err)
	}
	if t.OpenInterestValue, err = parseDecimal(d.OpenInterestValue); err != nil {
		return t, fmt.Errorf("OpenInterestValue: %w", err)
	}

	return t, nil
}

// parseDecimal безопасно парсит строку в decimal, возвращая 0 для пустых строк
func parseDecimal(s string) (decimal.Decimal, error) {
	if s == "" {
		return decimal.Zero, nil
	}
	return decimal.NewFromString(s)
}
