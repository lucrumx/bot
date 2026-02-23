package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// OrderSide represents the side of an order, either buy or sell, typically used in trading operations.
type OrderSide string

const (
	// OrderSideBuy represents the buy side of an order.
	OrderSideBuy OrderSide = "BUY"
	// OrderSideSell represents the sell side of an order.
	OrderSideSell OrderSide = "SELL"
)

// OrderType represents the type of order, either limit or market
type OrderType string

const (
	// OrderTypeLimit represents a limit order.
	OrderTypeLimit OrderType = "LIMIT"
	// OrderTypeMarket represents a market order.
	OrderTypeMarket OrderType = "MARKET"
)

// OrderStatus represents the status of an order, such as "NEW", "PARTIALLY_FILLED", "FILLED", etc.
type OrderStatus string

const (

	// OrderStatusNew represents an order status indicating that the order has been newly created and not yet processed.
	OrderStatusNew OrderStatus = "NEW"
	// OrderStatusPartiallyFilled represents an order status indicating that the order has been partially filled.
	OrderStatusPartiallyFilled OrderStatus = "PARTIALLY_FILLED"
	// OrderStatusFilled represents an order status indicating that the order has been fully filled (executed).
	OrderStatusFilled OrderStatus = "FILLED"
	// OrderStatusCanceled represents an order status indicating that the order has been canceled.
	OrderStatusCanceled OrderStatus = "CANCELED"
	// OrderStatusPending represents an order status indicating that the order is pending (not yet processed).
	OrderStatusPending OrderStatus = "PENDING"
	// OrderStatusRejected represents an order status indicating that the order has been rejected.
	OrderStatusRejected OrderStatus = "REJECTED"
	// OrderStatusExpired represents an order status indicating that the order has expired.
	OrderStatusExpired OrderStatus = "EXPIRED"
)

// OrderMarket represents the market type of an order, either spot or linear.
type OrderMarket string

const (
	// OrderMarketSpot represents a spot market type for an order.
	OrderMarketSpot OrderMarket = "SPOT"
	// OrderMarketLinear represents a linear (perpetual futures) market type for an order.
	OrderMarketLinear OrderMarket = "LINEAR"
)

// Order represents an order placed on an exchange.
type Order struct {
	ID               uuid.UUID       `gorm:"type:uuid;primary_key;default:uuidv7()"`
	ExchangeOrderID  string          `gorm:"type:text;"` // Exchange assigned order id
	ExchangeName     string          `gorm:"type:text;not null"`
	Symbol           string          `gorm:"type:text;not null"`
	Market           OrderMarket     `gorm:"type:text;not null"`
	Side             OrderSide       `gorm:"type:text;not null"`
	Type             OrderType       `gorm:"type:text;not null"`
	Price            decimal.Decimal `gorm:"type:decimal(28,12);"` // Price for limit order
	Quantity         decimal.Decimal `gorm:"type:decimal(28,12);not null"`
	AvgPrice         decimal.Decimal `gorm:"type:decimal(28,12);"` // Average price of the market order
	ExecutedQuantity decimal.Decimal `gorm:"type:decimal(28,12);"` // executed quantity for limit order
	Commission       decimal.Decimal `gorm:"type:decimal(28,12);"` // commission for the order
	HasErrors        bool            `gorm:"type:boolean;default:false"`
	RawResponse      string          `gorm:"type:text;"`
	CreatedAt        time.Time       `gorm:"type:timestamptz;default:now()"`
	UpdatedAt        time.Time       `gorm:"type:timestamptz;"`
}
