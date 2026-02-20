// Package exchange provides an interface for interacting with various cryptocurrency exchanges.
package exchange

import (
	"context"

	"github.com/lucrumx/bot/internal/models"
)

// Category represents the type of trading instruments available on an exchange.
type Category string

const (

	// CategorySpot represents the category for spot trading instruments on an ByBit exchange.
	CategorySpot Category = "spot"
	// CategoryLinear represents the category for linear trading instruments on an ByBit exchange.
	CategoryLinear Category = "linear"
)

// Provider represents an exchange provider (ByBit, Binance, BingX, and etc.).
type Provider interface {
	GetExchangeName() string
	GetTickers(ctx context.Context, symbols []string, category Category) ([]Ticker, error)
	SubscribeTrades(ctx context.Context, symbols []string) (<-chan Trade, error)
	CreateOrder(ctx, req models.Order) (*models.Order, error)
	GetBalances(ctx context.Context) ([]models.Balance, error)
	SetLeverage(ctx context.Context, symbol string, leverage int64) error
}
