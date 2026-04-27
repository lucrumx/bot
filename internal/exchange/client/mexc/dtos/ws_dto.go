package dtos

// WSBaseMessage reprsent base message for websockt event
type WSBaseMessage struct {
	Channel string `json:"channel"`
}

// WSTradeDTO represent trade data
type WSTradeDTO struct {
	Channel string  `json:"channel"`
	Symbol  string  `json:"symbol"`
	Ts      float64 `json:"ts"`
	Data    []struct {
		Price         float64 `json:"p"`
		Quantity      float64 `json:"v"`
		TradeSide     int64   `json:"T"` // Trade side: 1 buy, 2 sell
		O             int64   `json:"o"` // Open/close flag: 1 new position, 2 reduce position, 3 position unchanged. If O=1, v is the added position size
		M             int64   `json:"m"` // Self-trade: 1 yes, 2 no
		TransactionID string  `json:"i"` // Transaction ID
		TradeTime     int64   `json:"t"` // Trade time
	} `json:"data"`
}
