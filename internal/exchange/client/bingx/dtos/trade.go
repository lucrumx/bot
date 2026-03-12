package dtos

import "github.com/lucrumx/bot/internal/utils"

// WsTradeDataDTO represents a trade data transfer object.
type WsTradeDataDTO struct {
	T      int64             `json:"T"` // Trade time
	Volume utils.JSONFloat64 `json:"q"` // Volume quantity
	Price  utils.JSONFloat64 `json:"p"` // price
	M      bool              `json:"m"` // Whether the buyer is a market maker. If true, this trade is a passive sell order; otherwise, it is a passive buy order.
	Symbol string            `json:"s"` //Trading pair
}

// WsTradeMessageDTO represents a trade message transfer object.
type WsTradeMessageDTO struct {
	Code     int              `json:"code"`     // Error code, 0 for normal, 1 for error
	DataType string           `json:"dataType"` // Subscribed data type, e.g., BTC-USDT@trade
	Data     []WsTradeDataDTO `json:"data"`
}
