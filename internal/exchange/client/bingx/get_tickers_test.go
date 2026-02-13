package bingx

import (
	"testing"

	"github.com/stretchr/testify/assert"

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
