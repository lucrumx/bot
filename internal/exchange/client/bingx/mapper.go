package bingx

import (
	"github.com/lucrumx/bot/internal/exchange"
	"github.com/lucrumx/bot/internal/exchange/client/bingx/dtos"
)

func mapTicker(d dtos.TickerDTO) (exchange.Ticker, error) {
	var t exchange.Ticker

	t.Symbol = normalizeTickerName(d.Symbol)

	return t, nil
}
