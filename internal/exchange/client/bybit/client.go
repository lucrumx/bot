// Package bybit provides a client for the ByBit exchange.
package bybit

import (
	"context"
	"net/http"

	"github.com/lucrumx/bot/internal/exchange"
	"github.com/lucrumx/bot/internal/utils"
)

// Client represents a ByBit client.
type Client struct {
	baseURL   string
	http      *http.Client
	wsManager *WSManager
}

// NewByBitClient creates a new ByBitClient.
func NewByBitClient() *Client {
	return &Client{
		baseURL:   utils.GetEnv("BYBIT_BASE_URL", ""),
		http:      &http.Client{},
		wsManager: NewWSManager(),
	}
}

// SubscribeTrades initiates WebSocket trade subscriptions for the given symbols and streams trades to the returned channel.
func (c *Client) SubscribeTrades(ctx context.Context, symbols []string) (<-chan exchange.Trade, error) {
	return c.wsManager.SubscribeTrades(ctx, symbols)
}
