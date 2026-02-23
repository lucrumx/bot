// Package dtos contains data transfer objects for the BingX exchange.
package dtos

import "github.com/lucrumx/bot/internal/utils"

// Balance represents a balance on the exchange in asset (currency).
type Balance struct {
	Asset            string            `json:"asset"`   // currency
	Balance          utils.JSONFloat64 `json:"balance"` // deposit ± realized PnL ± fees ± funding
	Equity           utils.JSONFloat64 `json:"equity"`  // balance with unrealized profit: balance + unrealized PnL
	UnrealizedProfit utils.JSONFloat64 `json:"unrealizedProfit"`
	RealisedProfit   utils.JSONFloat64 `json:"realisedProfit"`
	AvailableMargin  utils.JSONFloat64 `json:"availableMargin"`
	UsedMargin       utils.JSONFloat64 `json:"usedMargin"`
	FreezedMargin    utils.JSONFloat64 `json:"freezedMargin"`
}

// ResponseGetBalanceDTO represents a response to a balance request.
type ResponseGetBalanceDTO struct {
	Code int64     `json:"code"`
	Msg  string    `json:"msg"`
	Data []Balance `json:"data"`
}
