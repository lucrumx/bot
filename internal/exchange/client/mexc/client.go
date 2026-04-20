// Package mexc provides a client for the MEXC exchange API.
package mexc

import (
	"context"
	"net/http"

	"github.com/rs/zerolog"

	"github.com/lucrumx/bot/internal/config"
	"github.com/lucrumx/bot/internal/exchange"
)

// Client represents a MEXC exchange client.
type Client struct {
	exchangeName string
	baseURL      string
	httpClient   *http.Client
	cfg          *config.Config
	logger       zerolog.Logger
	wsManager    *exchange.WSManager
}

// NewClient constructor.
func NewClient(cfg *config.Config, logger zerolog.Logger) *Client {
	return &Client{
		exchangeName: "MEXC",
		baseURL:      cfg.Exchange.MEXC.APIBaseURL,
		httpClient:   &http.Client{},
		cfg:          cfg,
		logger:       logger,
		wsManager: exchange.NewWSManager(cfg, func(c *config.Config) exchange.WsClient {
			return newWsClient(c, logger)
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

// SubscribeExecutions is not yet implemented for MEXC.
func (c *Client) SubscribeExecutions(_ context.Context) (<-chan exchange.OrderExecutionEvent, error) {
	return nil, nil
}
