package mexc

import (
	"fmt"
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/lucrumx/bot/internal/config"
)

func Test_GetBalanceIntegration(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("skip: set INTEGRATION_TEST=1 to run")
	}

	cfg := config.Config{
		Exchange: config.ExchangeConfig{
			MEXC: config.MEXCConfig{
				APIBaseURL: "https://api.mexc.com",
				APIKey:     "mx0vglFQVaOBJvUVWs",
				APISecret:  "8a9b5bf00ba444029642cb72082737ac",
			},
		},
	}

	logger := zerolog.Nop()
	ctx := t.Context()

	mexc := NewClient(&cfg, logger)

	balance, err := mexc.GetBalances(ctx)

	require.NoError(t, err)
	require.NotNil(t, balance)
	require.NotEmpty(t, balance)

	fmt.Println(balance)
}
