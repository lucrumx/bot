package mexc

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/lucrumx/bot/internal/config"
	"github.com/lucrumx/bot/internal/exchange"
)

func Test_WSClientIntegration(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("skip: set INTEGRATION_TEST=1 to run")
	}

	logger := zerolog.Nop()
	ctx := t.Context()
	ctx, cancel := context.WithCancel(ctx)
	go func() {
		time.Sleep(60 * time.Second)
		cancel()
	}()

	cfg := config.Config{
		Exchange: config.ExchangeConfig{
			MEXC: config.MEXCConfig{
				WSUrl: "wss://contract.mexc.com/edge",
			},
		},
	}

	symbols := []string{"TONUSDT", "ETHUSDT"}
	tradeChan := make(chan exchange.Trade, 100)

	mexcWSClient := newWsClient(&cfg, logger)
	err := mexcWSClient.Start(ctx, symbols, exchange.CategoryLinear, tradeChan)
	require.NoError(t, err)

	for {
		select {
		case <-ctx.Done():
			return
		case trade, ok := <-tradeChan:
			if !ok {
				return
			}
			t.Logf("trade: %+v", trade)
		}
	}
}
