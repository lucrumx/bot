// Package bybit provides a client for the ByBit exchange.
package bybit

import (
	"context"
	"net/http"

	"github.com/lucrumx/bot/internal/config"
	"github.com/lucrumx/bot/internal/exchange"
	"github.com/rs/zerolog"
)

// Client represents a ByBit client.
type Client struct {
	exchangeName string
	baseURL      string
	http         *http.Client
	cfg          *config.Config
	logger       zerolog.Logger

	wsManager        *exchange.WSManager
	wsPrivate        *WsPrivateClient
	wsPrivateStarted bool
}

// NewByBitClient creates a new ByBitClient.
func NewByBitClient(cfg *config.Config, logger zerolog.Logger) *Client {
	return &Client{
		exchangeName: "ByBit",
		baseURL:      cfg.Exchange.ByBit.BaseURL,
		http:         &http.Client{},
		cfg:          cfg,
		logger:       logger,
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
	if !c.wsPrivateStarted {
		c.wsPrivate = NewWsPrivateClient(c.cfg, c.logger)
		if err := c.wsPrivate.Start(ctx); err != nil {
			return nil, err
		}
		c.wsPrivateStarted = true
	}
	return c.wsPrivate.SubscribeToExecutions()
}
