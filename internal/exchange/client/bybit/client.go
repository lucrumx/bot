// Package bybit provides a client for the ByBit exchange.
package bybit

import (
	"context"
	"net/http"

	"github.com/lucrumx/bot/internal/config"
	"github.com/lucrumx/bot/internal/exchange"
)

// Client represents a ByBit client.
type Client struct {
	exchangeName string
	baseURL      string
	http         *http.Client
	wsManager    *exchange.WSManager
	cfg          *config.Config
}

// NewByBitClient creates a new ByBitClient.
func NewByBitClient(cfg *config.Config) *Client {
	return &Client{
		exchangeName: "ByBit",
		baseURL:      cfg.Exchange.ByBit.BaseURL,
		http:         &http.Client{},
		cfg:          cfg,
		wsManager: exchange.NewWSManager(cfg, func(c *config.Config) exchange.WsClient {
			return newWsClient(c)
		}),
	}
}

// GetExchangeName returns the exchange name.
func (c *Client) GetExchangeName() string {
	return c.exchangeName
}

// SubscribeTrades initiates WebSocket trade subscriptions for the given symbols and streams trades to the returned channel.
func (c *Client) SubscribeTrades(ctx context.Context, symbols []string) (<-chan exchange.Trade, error) {
	return c.wsManager.SubscribeTrades(ctx, symbols)
}
