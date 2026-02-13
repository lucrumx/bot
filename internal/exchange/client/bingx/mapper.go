package bingx

import (
	"github.com/lucrumx/bot/internal/exchange"
)

func mapTicker(d TickerDTO) (exchange.Ticker, error) {
	var t exchange.Ticker

	t.Symbol = d.Symbol

	return t, nil
}
