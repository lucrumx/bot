package dtos

import "github.com/shopspring/decimal"

// Order state constants for push.personal.order channel.
const (
	OrderStatePending  = 1
	OrderStateOpen     = 2
	OrderStateFilled   = 3
	OrderStateCanceled = 4
	OrderStateInvalid  = 5
)

// WSOrderDTO represents the push.personal.order message from MEXC private WebSocket.
type WSOrderDTO struct {
	Channel string    `json:"channel"`
	Data    OrderData `json:"data"`
	Ts      int64     `json:"ts"`
}

// OrderData contains order state details from the push.personal.order channel.
type OrderData struct {
	OrderID      string          `json:"orderId"`
	Symbol       string          `json:"symbol"`
	PositionID   int64           `json:"positionId"`
	Price        decimal.Decimal `json:"price"`
	Vol          decimal.Decimal `json:"vol"`
	Leverage     int64           `json:"leverage"`
	Side         int             `json:"side"`
	Category     int             `json:"category"`
	OrderType    int             `json:"orderType"`
	DealAvgPrice decimal.Decimal `json:"dealAvgPrice"`
	DealVol      decimal.Decimal `json:"dealVol"`
	OrderMargin  decimal.Decimal `json:"orderMargin"`
	TakerFee     decimal.Decimal `json:"takerFee"`
	MakerFee     decimal.Decimal `json:"makerFee"`
	Profit       decimal.Decimal `json:"profit"`
	FeeCurrency  string          `json:"feeCurrency"`
	OpenType     int             `json:"openType"`
	State        int             `json:"state"`
	ErrorCode    int             `json:"errorCode"`
	ExternalOid  string          `json:"externalOid"`
	CreateTime   int64           `json:"createTime"`
	UpdateTime   int64           `json:"updateTime"`
	RemainVol    decimal.Decimal `json:"remainVol"`
	PositionMode int             `json:"positionMode"`
	ReduceOnly   bool            `json:"reduceOnly"`
}
