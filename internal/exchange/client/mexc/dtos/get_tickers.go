// Package dtos contains data transfer objects for the MEXC exchange.
package dtos

// GetTickersDTO represent tickers data
type GetTickersDTO struct {
	Success bool        `json:"success"`
	Code    int         `json:"code"`
	Data    []TickerDTO `json:"data"`
}

// TickerDTO represent ticker in mexc get ticker data array
type TickerDTO struct {
	ContractID    int     `json:"contractId"`
	Symbol        string  `json:"symbol"`
	LastPrice     float64 `json:"lastPrice"`
	Bid1          float64 `json:"bid1"`
	Ask1          float64 `json:"ask1"`
	Volume24      float64 `json:"volume24"`
	Amount24      float64 `json:"amount24"`
	HoldVol       float64 `json:"holdVol"`
	Lower24Price  float64 `json:"lower24Price"`
	High24Price   float64 `json:"high24Price"`
	RiseFallRate  float64 `json:"riseFallRate"`
	RiseFallValue float64 `json:"riseFallValue"`
	IndexPrice    float64 `json:"indexPrice"`
	FairPrice     float64 `json:"fairPrice"`
	FundingRate   float64 `json:"fundingRate"`
	MaxBidPrice   float64 `json:"maxBidPrice"`
	MinAskPrice   float64 `json:"minAskPrice"`
	Timestamp     int64   `json:"timestamp"`
	// RiseFallRates          map[string]interface{} `json:"riseFallRates"`
	// RiseFallRatesOfTimezone []float64 `json:"riseFallRatesOfTimezone"`
}
