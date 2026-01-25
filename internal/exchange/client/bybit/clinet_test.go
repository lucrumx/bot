package bybit

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/lucrumx/bot/internal/exchange"
)

func TestClient_GetTickers_Integration(t *testing.T) {
	t.Setenv("BYBIT_BASE_URL", "https://api.bybit.nl")

	client := NewByBitClient()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tickers, err := client.GetTickers(ctx, []string{}, exchange.CategoryLinear)

	require.NoError(t, err)
	require.NotNil(t, tickers)
	require.NotEmpty(t, tickers)

	firstTicker := (*tickers)[0]

	log.Printf("First ticker symbol: %s, price: %s", firstTicker.Symbol, firstTicker.LastPrice)
}
