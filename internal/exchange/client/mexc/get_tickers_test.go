package mexc

import (
	"fmt"
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/lucrumx/bot/internal/config"
	"github.com/lucrumx/bot/internal/exchange"
)

func Test_GetTickers(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("skip: set INTEGRATION_TEST=1 to run")
	}

	logger := zerolog.Nop()
	ctx := t.Context()

	cfg := config.Config{
		Exchange: config.ExchangeConfig{
			MEXC: config.MEXCConfig{
				APIBaseURL: "https://api.mexc.com",
			},
		},
	}

	moexc := NewClient(&cfg, logger)

	tickers, err := moexc.GetTickers(ctx, []string{}, exchange.CategoryLinear)
	require.NoError(t, err)
	require.NotNil(t, tickers)
	require.NotEmpty(t, tickers)

	fmt.Print(tickers)
}
