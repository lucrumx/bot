package bybit

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lucrumx/bot/internal/config"
)

func TestClient_GetBalances(t *testing.T) {
	t.Skip("Skip integration test")

	cfg := config.Config{
		Exchange: config.ExchangeConfig{
			ByBit: config.ByBitConfig{
				APIKey:    "some-api-key",
				APISecret: "some-api-secret",
				BaseURL:   "https://api.bybit.nl",
			},
		},
	}

	client := NewByBitClient(&cfg)
	balances, err := client.GetBalances(t.Context())

	assert.NoError(t, err)
	assert.NotEmpty(t, balances)

	// pretty, err := json.MarshalIndent(balances, "", "  ")
	// t.Log("\n" + string(pretty))
}
