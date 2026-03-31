package dtos

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ExecutionDTO represent data for execution topic
type ExecutionDTO struct {
	Category   string `json:"category"`
	Symbol     string `json:"symbol"`
	ClosedSize string `json:"closedSize"`
	ExecFee    string `json:"execFee"`
	ExecID     string `json:"execId"`
	// Execution price
	ExecPrice decimal.Decimal `json:"execPrice"`
	// Execution quantity
	ExecQty decimal.Decimal `json:"execQty"`
	// Leaves quantity
	LeavesQty       decimal.Decimal `json:"leavesQty"`
	ExecType        string          `json:"execType"`
	ExecValue       decimal.Decimal `json:"execValue"`
	FeeRate         string          `json:"feeRate"`
	TradeIv         string          `json:"tradeIv"`
	MarkIv          string          `json:"markIv"`
	BlockTradeID    string          `json:"blockTradeId"`
	MarkPrice       string          `json:"markPrice"`
	IndexPrice      string          `json:"indexPrice"`
	UnderlyingPrice string          `json:"underlyingPrice"`
	OrderID         string          `json:"orderId"`
	// customer order id
	OrderLinkID uuid.UUID `json:"orderLinkId"`
	//
	OrderPrice    decimal.Decimal `json:"orderPrice"`
	OrderQty      decimal.Decimal `json:"orderQty"`
	OrderType     string          `json:"orderType"`
	StopOrderType string          `json:"stopOrderType"`
	Side          string          `json:"side"`
	ExecTime      string          `json:"execTime"`
	IsLeverage    string          `json:"isLeverage"`
	IsMaker       bool            `json:"isMaker"`
	Seq           int64           `json:"seq"`
	MarketUnit    string          `json:"marketUnit"`
	ExecPnl       string          `json:"execPnl"`
	CreateType    string          `json:"createType"`
	ExtraFees     []struct {
		FeeCoin    string `json:"feeCoin"`
		FeeType    string `json:"feeType"`
		SubFeeType string `json:"subFeeType"`
		FeeRate    string `json:"feeRate"`
		Fee        string `json:"fee"`
	} `json:"extraFees"`
	FeeCurrency string `json:"feeCurrency"`
}
