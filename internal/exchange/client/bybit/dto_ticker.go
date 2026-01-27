package bybit

// TickerDTO represents a Bybit ticker.
type TickerDTO struct {
	Symbol            string `json:"symbol"`
	LastPrice         string `json:"lastPrice"`
	IndexPrice        string `json:"indexPrice"`
	MarkPrice         string `json:"markPrice"`
	PrevPrice24h      string `json:"prevPrice24h"`
	Price24hPcnt      string `json:"price24hPcnt"`
	HighPrice24h      string `json:"highPrice24h"`
	LowPrice24h       string `json:"lowPrice24h"`
	PrevPrice1h       string `json:"prevPrice1h"`
	OpenInterest      string `json:"openInterest"`
	OpenInterestValue string `json:"openInterestValue"`
	Volume1m          string `json:"volume1m"`
	Volume5m          string `json:"volume5m"`
	Volume15m         string `json:"volume15m"`
	Turnover24h       string `json:"turnover24h"`
}
