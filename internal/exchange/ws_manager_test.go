package exchange

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/lucrumx/bot/internal/config"
)

type mockWsClient struct {
	symbols []string
}

func (m *mockWsClient) Start(ctx context.Context, symbols []string, outChan chan<- Trade) error {
	m.symbols = symbols
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				for _, s := range symbols {
					outChan <- Trade{
						Symbol: s,
						Side:   Buy,
						Price:  100.50,
						Volume: 0.1,
						Ts:     time.Now().UnixMilli(),
					}
				}
			}
		}
	}()
	return nil
}

func TestWSManager_SubscribeTrades(t *testing.T) {
	cfg := config.Config{
		Exchange: config.ExchangeConfig{
			WsClient: config.WsClientConfig{
				BufferSize: 5000,
			},
		},
	}

	mockClientFactory := func(_ *config.Config) WsClient {
		return &mockWsClient{}
	}

	manager := NewWSManager(&cfg, mockClientFactory)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	symbols := []string{"BTCUSDT", "ETHUSDT", "TONUSDT"}
	tradeChannel, err := manager.SubscribeTrades(ctx, symbols)

	require.NoError(t, err)
	require.NotNil(t, tradeChannel)

	cnt := 0

	//for trade := range tradeChannel {
	for range tradeChannel {
		cnt++

		if cnt >= 10 {
			log.Printf("Received %d trades. Test passed.", cnt)
			break
		}
	}

	require.Greater(t, cnt, 0)
}
