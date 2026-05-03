package dtos

import (
	"github.com/lucrumx/bot/internal/utils"
)

// GetOrderResponseDTO — ответ от MEXC Futures API.
type GetOrderResponseDTO struct {
	Success utils.FlexibleBool `json:"success"`
	Code    int                `json:"code"`
	Data    struct {
		OrderID                  string        `json:"orderId"`
		Symbol                   string        `json:"symbol"`
		PositionID               int64         `json:"positionId"`
		Price                    utils.Decimal `json:"price"`
		PriceStr                 utils.Decimal `json:"priceStr"`
		Vol                      utils.Decimal `json:"vol"`
		Leverage                 int           `json:"leverage"`
		Side                     int           `json:"side"`
		Category                 int           `json:"category"`
		OrderType                int           `json:"orderType"`
		DealAvgPrice             utils.Decimal `json:"dealAvgPrice"`
		DealAvgPriceStr          utils.Decimal `json:"dealAvgPriceStr"`
		DealVol                  utils.Decimal `json:"dealVol"`
		OrderMargin              utils.Decimal `json:"orderMargin"`
		TakerFee                 utils.Decimal `json:"takerFee"`
		MakerFee                 utils.Decimal `json:"makerFee"`
		Profit                   utils.Decimal `json:"profit"`
		FeeCurrency              string        `json:"feeCurrency"`
		OpenType                 int           `json:"openType"`
		State                    int           `json:"state"`
		ExternalOID              string        `json:"externalOid"`
		ErrorCode                int           `json:"errorCode"`
		UsedMargin               utils.Decimal `json:"usedMargin"`
		CreateTime               utils.Time    `json:"createTime"`
		UpdateTime               utils.Time    `json:"updateTime"`
		PositionMode             int           `json:"positionMode"`
		Version                  int           `json:"version"`
		ShowCancelReason         int           `json:"showCancelReason"`
		ShowProfitRateShare      int           `json:"showProfitRateShare"`
		BBOTypeNum               int           `json:"bboTypeNum"`
		TotalFee                 utils.Decimal `json:"totalFee"`
		ZeroSaveTotalFeeBinance  utils.Decimal `json:"zeroSaveTotalFeeBinance"`
		ZeroTradeTotalFeeBinance utils.Decimal `json:"zeroTradeTotalFeeBinance"`
	} `json:"data"`
}
