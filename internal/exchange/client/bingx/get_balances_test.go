package bingx

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lucrumx/bot/internal/config"
)

func TestClient_GetBalances(t *testing.T) {
	t.Skip("Skip integration test")

	cfg := &config.Config{
		Exchange: config.ExchangeConfig{
			BingX: config.BingXConfig{
				APIKey:    "some-api-key",
				APISecret: "some-api-secret",
			},
		},
	}

	bingx := NewClient(cfg)

	ctx := t.Context()
	balances, err := bingx.GetBalances(ctx)

	assert.NoError(t, err)
	assert.NotEmpty(t, balances)

	fmt.Printf("%+v\n", balances)
}
