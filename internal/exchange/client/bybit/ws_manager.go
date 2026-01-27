package bybit

import (
	"context"
	"fmt"
	"log"
	"strconv"

	"github.com/lucrumx/bot/internal/exchange"
	"github.com/lucrumx/bot/internal/utils"
)

const chunkSize = 100

// WSManager manages multiple WebSocket clients for handling trade subscriptions and streaming data.
type WSManager struct {
	wsClient []*wsClient
}

// NewWSManager initializes and returns a new instance of WSManager with an empty list of wsClient.
func NewWSManager() *WSManager {
	return &WSManager{
		wsClient: make([]*wsClient, 0),
	}
}

// SubscribeTrades initiates WebSocket trade subscriptions for the given symbols and streams trades to the returned channel.
func (m *WSManager) SubscribeTrades(ctx context.Context, symbols []string) (<-chan exchange.Trade, error) {
	bufferSize, err := strconv.Atoi(utils.GetEnv("WS_CLIENT_BUFFER_SIZE", "1000"))
	if err != nil {
		log.Fatalf("Cannot get env WS_CLIENT_BUFFER_SIZE: %v", err)
	}

	outChan := make(chan exchange.Trade, bufferSize)

	symbolsChunks := chunkSymbols(symbols)

	for _, chunk := range symbolsChunks {
		wsClient := newWsClient()
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
