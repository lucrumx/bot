package exchange

import "github.com/shopspring/decimal"

// Ticker represents market data for a specific trading instrument or asset.
type Ticker struct {
	Symbol string

	// Цена последнего исполненного трейда
	LastPrice decimal.Decimal

	// (!) для информации, и рассчитывается биржей, агрегируется с нескольких торговых площадок
	IndexPrice decimal.Decimal

	// Используется для расчета нереализованного PnL, ликвидаций, сглажена, не реагирует резко на шпильки,
	// ликвидация происходит по markPrice
	MarkPrice decimal.Decimal

	// lastPrice 24h назад
	PrevPrice24h decimal.Decimal

	// Процент изменения за 24h, (lastPrice - prevPrice24h) / prevPrice24h
	Price24hPcnt decimal.Decimal

	// Максимальная цена за 24h
	HighPrice24h decimal.Decimal

	// Минимальная цена за 24h
	LowPrice24h decimal.Decimal

	// Цена час назад
	PrevPrice1h decimal.Decimal

	// Количество открытых контрактов
	OpenInterest decimal.Decimal

	// Денежная стоимость открытых контрактов
	OpenInterestValue decimal.Decimal

	// Оборот за 24h
	Turnover24h decimal.Decimal

	// TODO Add this fields. They present in bingx get ticker response and should be analog in others
	// MakerFeeRate      float64 `json:"makerFeeRate"`
	//	TakerFeeRate      float64 `json:"takerFeeRate"`
}
