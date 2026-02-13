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
	baseURL    string
	httpClient *http.Client
	cfg        *config.Config
	wsManager  *exchange.WSManager
}

// NewClient constructor.
func NewClient(cfg *config.Config) *Client {
	return &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{},
		cfg:        cfg,
		wsManager: exchange.NewWSManager(cfg, func(c *config.Config) exchange.WsClient {
			return newWsClient(c)
		}),
	}
}

// SubscribeTrades initiates WebSocket trade subscriptions for the given symbols and streams trades to the returned channel.
func (c *Client) SubscribeTrades(ctx context.Context, symbols []string) (<-chan exchange.Trade, error) {
	return c.wsManager.SubscribeTrades(ctx, symbols)
}
