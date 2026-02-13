// Package bingx provides a client for the Bing-X exchange.
package bingx

import "github.com/lucrumx/bot/internal/utils"

// TickerDTO represents a ticker data transfer object.
type TickerDTO struct {
	Symbol            string  `json:"symbol"`
	QuantityPrecision int64   `json:"quantityPrecision"`
	PricePrecision    int64   `json:"pricePrecision"`
	MakerFeeRate      float64 `json:"makerFeeRate"`
	TakerFeeRate      float64 `json:"takerFeeRate"`
	TriggerFeeRate    string  `json:"triggerFeeRate"`
	Status            int64   `json:"status"` // 1 online, 25 forbidden to open positions, 5 pre-online, 0 offline
}

// ResponseGetTickerDTO represents a response to a ticker request.
type ResponseGetTickerDTO struct {
	Code int64       `json:"code"`
	Msg  string      `json:"msg"`
	Data []TickerDTO `json:"data"`
}

// WsTradeDataDTO represents a trade data transfer object.
type WsTradeDataDTO struct {
	T      int64             `json:"T"`        // Trade time
	Volume utils.JSONFloat64 `json:"quantity"` // Volume quantity
	Price  utils.JSONFloat64 `json:"price"`    // price
	M      bool              `json:"m"`        // Whether the buyer is a market maker. If true, this trade is a passive sell order; otherwise, it is a passive buy order.
	Symbol string            `json:"s"`        //Trading pair
}

// WsTradeMessageDTO represents a trade message transfer object.
type WsTradeMessageDTO struct {
	Code     int              `json:"code"`     // Error code, 0 for normal, 1 for error
	DataType string           `json:"dataType"` // Subscribed data type, e.g., BTC-USDT@trade
	Data     []WsTradeDataDTO `json:"data"`
}
