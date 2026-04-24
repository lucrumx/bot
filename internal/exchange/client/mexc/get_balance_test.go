package mexc

import (
	"fmt"
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/lucrumx/bot/internal/utils/testutils"
)

func Test_GetBalanceIntegration(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("skip: set INTEGRATION_TEST=1 to run")
	}

	cfg := testutils.LoadTestConfig(t)

	logger := zerolog.Nop()
	ctx := t.Context()

	mexc := NewClient(cfg, logger)

	balance, err := mexc.GetBalances(ctx)

	require.NoError(t, err)
	require.NotNil(t, balance)
	require.NotEmpty(t, balance)

	fmt.Println(balance)
}
