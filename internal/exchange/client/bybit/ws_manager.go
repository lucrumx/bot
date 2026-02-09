package bybit

import (
	"context"
	"fmt"

	"github.com/lucrumx/bot/internal/config"
	"github.com/lucrumx/bot/internal/exchange"
)

const chunkSize = 100

// WSManager manages multiple WebSocket clients for handling trade subscriptions and streaming data.
type WSManager struct {
	wsClient []*wsClient
	cfg      *config.Config
}

// NewWSManager initializes and returns a new instance of WSManager with an empty list of wsClient.
func NewWSManager(cfg *config.Config) *WSManager {
	return &WSManager{
		wsClient: make([]*wsClient, 0),
		cfg:      cfg,
	}
}

// SubscribeTrades initiates WebSocket trade subscriptions for the given symbols and streams trades to the returned channel.
func (m *WSManager) SubscribeTrades(ctx context.Context, symbols []string) (<-chan exchange.Trade, error) {
	bufferSize := m.cfg.Exchange.WsClient.BufferSize

	outChan := make(chan exchange.Trade, bufferSize)

	symbolsChunks := chunkSymbols(symbols)

	for _, chunk := range symbolsChunks {
		wsClient := newWsClient(m.cfg)
		m.wsClient = append(m.wsClient, wsClient)

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
