package bybit

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"

	"github.com/lucrumx/bot/internal/config"
)

func TestWSManager_SubscribeTrades(t *testing.T) {
	t.Skip("Skipping integration test")
	t.Setenv("BYBIT_WS_BASE_URL", "wss://stream.bybit.com")

	cfg := &config.Config{
		Exchange: config.ExchangeConfig{
			WsClient: config.WsClientConfig{
				BufferSize: 5000,
			},
		},
	}

	manager := NewWSManager(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	symbols := []string{"BTCUSDT", "ETHUSDT", "TONUSDT"}
	streamChannel, err := manager.SubscribeTrades(ctx, symbols)

	require.NoError(t, err)
	require.NotNil(t, streamChannel)

	cnt := 0

	for trade := range streamChannel {
		cnt++

		log.Printf("[TRADE %d] %s | %s | price: %s | volume: %s",
			cnt,
			trade.Symbol,
			trade.Side,
			decimal.NewFromFloat(trade.Price).StringFixed(4),
			decimal.NewFromFloat(trade.Volume).StringFixed(4),
		)

		if cnt == 10 {
			log.Printf("Receive %d trades. Test passed. Stoping ...", cnt)
			break
		}
	}

	require.Greater(t, cnt, 0)
}
