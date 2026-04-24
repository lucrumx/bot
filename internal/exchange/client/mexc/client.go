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
	//
	createOrderURL string
	//
	wsPrivate        *WsPrivateClient
	wsPrivateStarted bool
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
		createOrderURL: cfg.Exchange.MEXC.APIBaseURL + createOrderURL,
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

// SubscribeExecutions subscribes to order execution events via private WebSocket.
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
