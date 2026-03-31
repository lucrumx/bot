// Package bingx provides a client for the Bing-X exchange.
package bingx

import (
	"context"
	"net/http"

	"github.com/lucrumx/bot/internal/config"
	"github.com/lucrumx/bot/internal/exchange"
)

const baseURL = "https://open-api.bingx.com"

// Client represents a BingX exchange client.
type Client struct {
	exchangeName string
	baseURL      string
	httpClient   *http.Client
	cfg          *config.Config
	wsManager    *exchange.WSManager
}

// NewClient constructor.
func NewClient(cfg *config.Config) *Client {
	return &Client{
		exchangeName: "BingX",
		baseURL:      baseURL,
		httpClient:   &http.Client{},
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
func (c *Client) SubscribeTrades(ctx context.Context, symbols []string, category exchange.Category) (<-chan exchange.Trade, error) {
	return c.wsManager.SubscribeTrades(ctx, symbols, category)
}

func (c *Client) SubscribeExecutions(ctx context.Context) (<-chan exchange.OrderExecutionEvent, error) {
	return nil, nil
}
