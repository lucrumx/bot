// Package dtos contains data transfer objects for the Bybit exchange.
package dtos

// OrderDTO represents an order in response.
type OrderDTO struct {
	OrderID     string `json:"orderId"`
	OrderLinkID string `json:"orderLinkId"`
}

// OrderCreateResponseDTO represents a response to a place order request.
type OrderCreateResponseDTO struct {
	RetCode    int      `json:"retCode"`
	RetMsg     string   `json:"retMsg"`
	Result     OrderDTO `json:"result"`
	RetExtInfo struct {
	} `json:"retExtInfo"`
	Time int64 `json:"time"`
}

// OrderSide represents the side of an order.
type OrderSide string

const (
	// OrderSideBuy represents a constant indicating a buy order.
	OrderSideBuy OrderSide = "Buy"
	// OrderSideSell represents a constant indicating a sell order.
	OrderSideSell OrderSide = "Sell"
)

type OrderType string

const (
	// OrderTypeLimit represents a constant indicating a limit order.
	OrderTypeLimit OrderType = "Limit"
	// OrderTypeMarket represents a constant indicating a market order.
	OrderTypeMarket OrderType = "Market"
)
