package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// OrderSide represents the side of an order, either buy or sell, typically used in trading operations.
type OrderSide string

const (
	// Buy represents the buy side of an order.
	Buy OrderSide = "BUY"
	// Sell represents the sell side of an order.
	Sell OrderSide = "SELL"
)

// OrderType represents the type of an order, either limit or market
type OrderType string

const (
	// Limit represents a limit order.
	Limit OrderType = "LIMIT"
	// Market represents a market order.
	Market OrderType = "MARKET"
)

// OrderStatus represents the status of an order, such as "NEW", "PARTIALLY_FILLED", "FILLED", etc.
type OrderStatus string

const (

	// New represents an order status indicating that the order has been newly created and not yet processed.
	New OrderStatus = "NEW"
	// PartiallyFilled represents an order status indicating that the order has been partially filled.
	PartiallyFilled OrderStatus = "PARTIALLY_FILLED"
	// Filled represents an order status indicating that the order has been fully filled (executed).
	Filled OrderStatus = "FILLED"
	// Canceled represents an order status indicating that the order has been canceled.
	Canceled OrderStatus = "CANCELED"
	// Pending represents an order status indicating that the order is pending (not yet processed).
	Pending OrderStatus = "PENDING"
	// Rejected represents an order status indicating that the order has been rejected.
	Rejected OrderStatus = "REJECTED"
	// Expired represents an order status indicating that the order has expired.
	Expired OrderStatus = "EXPIRED"
)

// Order represents an order placed on an exchange.
type Order struct {
	ID               uuid.UUID       `gorm:"type:uuid;primary_key;default:uuidv7()"`
	OrderID          string          `gorm:"type:text;not null"` // Exchange assigned order id
	ExchangeName     string          `gorm:"type:text;not null"`
	Symbol           string          `gorm:"type:text;not null"`
	ClientOrderID    uuid.UUID       `gorm:"type:uuid;default:uuidv7();index:idx_order_request_client_order_id"`
	Side             OrderSide       `gorm:"type:text;not null"`
	Type             OrderType       `gorm:"type:text;not null"`
	Price            decimal.Decimal `gorm:"type:decimal(28,12);"` // Price for limit order
	Quantity         decimal.Decimal `gorm:"type:decimal(28,12);not null"`
	AvgPrice         decimal.Decimal `gorm:"type:decimal(28,12);"` // Average price of the market order
	ExecutedQuantity decimal.Decimal `gorm:"type:decimal(28,12);"` // executed quantity for limit order
	Commission       decimal.Decimal `gorm:"type:decimal(28,12);"` // commission for the order
	CreatedAt        time.Time       `gorm:"type:timestampz;default:now()"`
	UpdatedAt        time.Time       `gorm:"type:timestampz;"`
}
