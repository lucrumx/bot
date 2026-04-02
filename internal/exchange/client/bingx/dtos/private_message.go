package dtos

import (
	"encoding/json"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// PrivateMessageDTO represents the structure of a message payload in BingX private websocket channel.
type PrivateMessageDTO struct {
	EventType string          `json:"e"`
	EventTS   int64           `json:"E"`
	TradeTS   int64           `json:"T"`
	O         json.RawMessage `json:"o"`  // https://bingx-api.github.io/docs-v3/#/en/Swap/Websocket%20Account%20Data/Order%20update%20push
	A         json.RawMessage `json:"a"`  // account update https://bingx-api.github.io/docs-v3/#/en/Swap/Websocket%20Account%20Data/Account%20balance%20and%20position%20update%20push
	AC        json.RawMessage `json:"ac"` // account configuration update such as leverage https://bingx-api.github.io/docs-v3/#/en/Swap/Websocket%20Account%20Data/Configuration%20updates%20such%20as%20leverage%20and%20margin%20mode
}

// ExecutionDTO represents a single execution data transfer object (order executions).
// https://bingx-api.github.io/docs-v3/#/en/Swap/Websocket%20Account%20Data/Order%20update%20push
type ExecutionDTO struct {
	//
	Symbol       string          `json:"s"`
	OrderID      uuid.UUID       `json:"c"` // client order id
	BingXOrderID int64           `json:"i"`
	Side         string          `json:"S"`  // Order side (BUY/SELL)
	OrderType    string          `json:"o"`  // Order type (LIMIT/MARKET, etc.)
	Qty          decimal.Decimal `json:"q"`  // order quantity
	Price        decimal.Decimal `json:"p"`  // order price
	Sp           decimal.Decimal `json:"sp"` // trigger price
	AvgPrice     decimal.Decimal `json:"ap"` // Average filled price
	X            string          `json:"x"`  // Execution type for this event (e.g., TRADE)
	OrderStatus  string          `json:"X"`  // // Current order status (NEW/PARTIALLY_FILLED/FILLED/CANCELED, etc.)
	Asset        string          `json:"N"`  // Fee asset (e.g., USDT)
	Fee          decimal.Decimal `json:"n"`  // Fee (may be negative)
	TradeTS      int64           `json:"T"`  // Trade timestamp (milliseconds since Unix epoch)
	Wt           string          `json:"wt"`
	Ps           string          `json:"ps"`
	PNL          string          `json:"rp"` // realized pnl
	FilledQty    decimal.Decimal `json:"z"`  // Cumulative filled quantity
	Sg           string          `json:"sg"`
	Ti           int             `json:"ti"`
	Ro           bool            `json:"ro"`
	TradeID      int             `json:"td"` // trade ID
	TradeValue   decimal.Decimal `json:"tv"` // Trade value / notional
}
