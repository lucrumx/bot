package dtos

import (
	"github.com/lucrumx/bot/internal/utils"
)

// GetOrderResponseDTO represents the response from the BingX get order endpoint.
type GetOrderResponseDTO struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Order struct {
			Symbol        string        `json:"symbol"`
			OrderID       int64         `json:"orderId"`
			Side          string        `json:"side"`
			PositionSide  string        `json:"positionSide"`
			Type          string        `json:"type"`
			OrigQty       utils.Decimal `json:"origQty"`
			Price         utils.Decimal `json:"price"`
			ExecutedQty   utils.Decimal `json:"executedQty"`
			AvgPrice      utils.Decimal `json:"avgPrice"`
			CumQuote      utils.Decimal `json:"cumQuote"`
			StopPrice     utils.Decimal `json:"stopPrice"`
			Profit        utils.Decimal `json:"profit"`
			Commission    utils.Decimal `json:"commission"`
			Status        string        `json:"status"`
			Time          utils.Time    `json:"time"`
			UpdateTime    utils.Time    `json:"updateTime"`
			ClientOrderID string        `json:"clientOrderId"`
			Leverage      string        `json:"leverage"`
			TakeProfit    struct {
				Type        string        `json:"type"`
				Quantity    utils.Decimal `json:"quantity"`
				StopPrice   utils.Decimal `json:"stopPrice"`
				Price       utils.Decimal `json:"price"`
				WorkingType string        `json:"workingType"`
			} `json:"takeProfit"`
			StopLoss struct {
				Type        string        `json:"type"`
				Quantity    utils.Decimal `json:"quantity"`
				StopPrice   utils.Decimal `json:"stopPrice"`
				Price       utils.Decimal `json:"price"`
				WorkingType string        `json:"workingType"`
			} `json:"stopLoss"`
			AdvanceAttr            int                `json:"advanceAttr"`
			PositionID             int64              `json:"positionID"`
			TakeProfitEntrustPrice utils.Decimal      `json:"takeProfitEntrustPrice"`
			StopLossEntrustPrice   utils.Decimal      `json:"stopLossEntrustPrice"`
			OrderType              string             `json:"orderType"`
			WorkingType            string             `json:"workingType"`
			StopGuaranteed         utils.FlexibleBool `json:"stopGuaranteed"`
			TriggerOrderID         int64              `json:"triggerOrderId"`
		} `json:"order"`
	} `json:"data"`
}
