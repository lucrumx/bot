package dtos

// OrderPositionSide represents a side of an order.
type OrderPositionSide string

const (
	// OrderPositionSideLong represents a constant indicating a long position in trading or financial operations.
	OrderPositionSideLong OrderPositionSide = "LONG"
	// OrderPositionSideShort represents a constant indicating a short position in trading or financial operations.
	OrderPositionSideShort OrderPositionSide = "SHORT"
)

// OrderSide represents a side of an order.
type OrderSide string

const (
	// OrderSideBuy represents a constant indicating a buy order.
	OrderSideBuy OrderSide = "BUY"
	// OrderSideSell represents a constant indicating a sell order.
	OrderSideSell OrderSide = "SELL"
)

// OrderType represents the type of order.
type OrderType string

const (
	// OrderTypeLimit represents a constant indicating a limit order.
	OrderTypeLimit OrderType = "LIMIT"
	// OrderTypeMarket represents a constant indicating a market order.
	OrderTypeMarket OrderType = "MARKET"
)

// OrderDTO represents an order in response.
type OrderDTO struct {
	Symbol  string `json:"symbol"`
	OrderID int64  `json:"orderId"`
	// OrderIDAlt тупорылый bingx шлет в ответе и orderId и orderID, причем второе - пустое и оно ломает unmarshal-инг, нужно его явно указать чтобы не падало
	OrderIDAlt    string `json:"orderID"`
	Side          string `json:"side"`
	PositionSide  string `json:"positionSide"`
	Type          string `json:"type"`
	ClientOrderID string `json:"clientOrderId"`
	WorkingType   string `json:"workingType"`
}

// OrderCreateResponseDTO represents a response to a place order request.
type OrderCreateResponseDTO struct {
	Code int64  `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		Order OrderDTO `json:"order"`
	} `json:"data"`
}
