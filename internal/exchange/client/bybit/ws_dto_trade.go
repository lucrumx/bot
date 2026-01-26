package bybit

import "github.com/lucrumx/bot/internal/exchange"

// wsTradeDTO represents a trade data in websocket trade message.
type wsTradeDTO struct {
	T      int64         `json:"T"` // The timestamp (ms) that the order is filled
	Symbol string        `json:"s"` // The symbol of the order
	Side   exchange.Side `json:"S"` // The side of the order (Buy or Sell)
	Volume string        `json:"v"` // Trade size
	Price  string        `json:"p"` // Trade price
}

// WsTradeMessageDTO WsTradeMessage represents a WebSocket trade message containing multiple trade details.
type wsTradeMessageDTO struct {
	Topic string       `json:"topic"`
	Typ   string       `json:"type"`
	Ts    string       `json:"ts"`
	Data  []wsTradeDTO `json:"data"`
}
