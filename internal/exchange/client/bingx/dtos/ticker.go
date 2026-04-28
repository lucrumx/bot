package dtos

import "encoding/json"

// TickerDTO represents a ticker data transfer object.
type TickerDTO struct {
	Symbol            string  `json:"symbol"`
	DisplayName       string  `json:"displayName"`
	Size              string  `json:"size"`
	QuantityPrecision int64   `json:"quantityPrecision"`
	PricePrecision    int64   `json:"pricePrecision"`
	MakerFeeRate      float64 `json:"makerFeeRate"`
	TakerFeeRate      float64 `json:"takerFeeRate"`
	TriggerFeeRate    string  `json:"triggerFeeRate"`
	Status            int64   `json:"status"` // 1 online, 25 forbidden to open positions, 5 pre-online, 0 offline
}

// ResponseGetTickerDTO represents a response to a ticker request.
// BingX returns Data as an array for multiple symbols, but as a single object for one symbol.
type ResponseGetTickerDTO struct {
	Code int64           `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

// ParseData handles both array and single object responses from BingX.
func (r *ResponseGetTickerDTO) ParseData() ([]TickerDTO, error) {
	if len(r.Data) == 0 {
		return nil, nil
	}
	// try array first
	var arr []TickerDTO
	if err := json.Unmarshal(r.Data, &arr); err == nil {
		return arr, nil
	}
	// fallback to single object
	var single TickerDTO
	if err := json.Unmarshal(r.Data, &single); err != nil {
		return nil, err
	}
	return []TickerDTO{single}, nil
}
