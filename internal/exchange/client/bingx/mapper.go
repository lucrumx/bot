package bingx

import (
	"strings"

	"github.com/lucrumx/bot/internal/exchange"
)

func mapTicker(d TickerDTO) (exchange.Ticker, error) {
	var t exchange.Ticker

	if strings.Contains(d.Symbol, "-USDT") {
		d.Symbol = strings.TrimSuffix(d.Symbol, "-USDT") + "USDT"
	}

	t.Symbol = d.Symbol

	return t, nil
}
