package exchange

import (
	"context"
	"fmt"

	"github.com/lucrumx/bot/internal/config"
)

// WsClient defines an interface for starting a websocket client to stream trades for specified symbols into an output channel.
type WsClient interface {
	Start(ctx context.Context, symbols []string, outChan chan<- Trade) error
}

// WSClientFactory defines a function type that creates a WsClient instance based on configuration.
type WSClientFactory func(cfg *config.Config) WsClient

const chunkSize = 100

// WSManager manages multiple WebSocket clients for handling trade subscriptions and streaming data.
type WSManager struct {
	wsClients []WsClient
	cfg       *config.Config
	factory   WSClientFactory
}

// NewWSManager initializes and returns a new instance of WSManager with an empty list of wsClient.
func NewWSManager(cfg *config.Config, factory WSClientFactory) *WSManager {
	return &WSManager{
		wsClients: make([]WsClient, 0),
		cfg:       cfg,
		factory:   factory,
	}
}

// SubscribeTrades initiates WebSocket trade subscriptions for the given symbols and streams trades to the returned channel.
func (m *WSManager) SubscribeTrades(ctx context.Context, symbols []string) (<-chan Trade, error) {
	bufferSize := m.cfg.Exchange.WsClient.BufferSize
	outChan := make(chan Trade, bufferSize)
	symbolsChunks := chunkSymbols(symbols)

	for _, chunk := range symbolsChunks {
		wsClient := m.factory(m.cfg)
		m.wsClients = append(m.wsClients, wsClient)

		if err := wsClient.Start(ctx, chunk, outChan); err != nil {
			return nil, fmt.Errorf("failed to start ws client: %w", err)
		}
	}

	go func() {
		<-ctx.Done()
		close(outChan)
	}()

	return outChan, nil
}

func chunkSymbols(symbols []string) [][]string {
	var chunks [][]string
	for i := 0; i < len(symbols); i += chunkSize {
		end := i + chunkSize
		if end > len(symbols) {
			end = len(symbols)
		}
		chunks = append(chunks, symbols[i:end])
	}
	return chunks
}
