package bybit

import (
	"os"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/lucrumx/bot/internal/utils/testutils"
)

func Test_GetInstruments_Integration(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "1" {
		t.Skip("skip: set INTEGRATION_TEST=1 to run")
	}

	cfg := testutils.LoadTestConfig(t)
	logger := zerolog.Nop()

	bybit := NewByBitClient(cfg, logger)

	intruments, err := bybit.GetInstruments(t.Context())
	require.NoError(t, err, "Error getting instruments")
	require.NotEmpty(t, intruments, "No instruments found")
	require.Contains(t, intruments, "TONUSDT", "Expected TONUSDT in instruments")

	testutils.PrintStruct(t, intruments["TONUSDT"], "TONUSDT Instrument")
}
