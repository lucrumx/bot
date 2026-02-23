package dtos

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
