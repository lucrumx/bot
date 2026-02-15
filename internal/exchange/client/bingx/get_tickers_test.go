package bingx

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lucrumx/bot/internal/exchange"

	"github.com/lucrumx/bot/internal/config"
)

func TestClient_GetTickers(t *testing.T) {
	t.Skip("Skip integration test")

	cfg := &config.Config{
		Exchange: config.ExchangeConfig{
			BingX: config.BingXConfig{
				APIKey:    "some-api-key",
				APISecret: "some-api-secret",
			},
		},
	}

	client := NewClient(cfg)
	ctx := t.Context()

	tickers, err := client.GetTickers(ctx, []string{}, exchange.CategoryLinear)

	assert.NoError(t, err)
	assert.NotEmpty(t, tickers)
}

func TestClient_GetTickers_NormalizesTickerSymbols(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/openApi/swap/v2/quote/contracts", r.URL.Path)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"code": 0,
			"msg": "",
			"data": [
				{"symbol":"BTC-USDT"},
				{"symbol":"ETH-USDT"}
			]
		}`))
	}))
	defer srv.Close()

	cfg := &config.Config{
		Exchange: config.ExchangeConfig{
			BingX: config.BingXConfig{
				APIKey:    "k",
				APISecret: "s",
			},
		},
	}

	c := NewClient(cfg)
	c.baseURL = srv.URL
	c.httpClient = srv.Client()

	tickers, err := c.GetTickers(context.Background(), []string{"BTCUSDT"}, exchange.CategoryLinear)
	require.NoError(t, err)
	require.Len(t, tickers, 2)

	// should be normalized
	require.Equal(t, "BTCUSDT", tickers[0].Symbol)
	require.Equal(t, "ETHUSDT", tickers[1].Symbol)
}
