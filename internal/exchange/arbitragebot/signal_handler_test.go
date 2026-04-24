package arbitragebot

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/lucrumx/bot/internal/models"
)

type notifierStub struct {
	msgs []string
}

func (n *notifierStub) Send(message string) error {
	n.msgs = append(n.msgs, message)
	return nil
}

type repoStub struct{}

func (r *repoStub) Create(_ context.Context, _ *models.ArbitrageSpread) error { return nil }
func (r *repoStub) Update(_ context.Context, _ *models.ArbitrageSpread, _ FindFilter) error {
	return nil
}
func (r *repoStub) FindAll(_ context.Context, _ FindFilter) ([]*models.ArbitrageSpread, error) {
	return nil, nil
}

func TestFormatPrice_PreservesLowPricePrecision(t *testing.T) {
	require.Equal(t, "0.00019400", formatPrice(0.000194))
	require.Equal(t, "0.00020000", formatPrice(0.0002))
	require.Equal(t, "1.2345", formatPrice(1.2345))
}

func TestSignalHandler_HandleNewSpreadEvent_ShowsDistinctLowPrices(t *testing.T) {
	notif := &notifierStub{}
	engine := newExecutionEngine(nil, newInstrumentCache(), zerolog.Nop())
	handler := newSignalHandler(notif, zerolog.Nop(), &repoStub{}, engine)

	handler.handleNewSpreadEvent(context.Background(), &SpreadEvent{
		Status:            models.ArbitrageSpreadOpened,
		Symbol:            "AKEUSDT",
		BuyOnExchange:     "ByBit",
		SellOnExchange:    "BingX",
		BuyPrice:          0.000194,
		SellPrice:         0.0002,
		FromSpreadPercent: 3.09,
		MaxSpreadPercent:  3.09,
	})

	require.Len(t, notif.msgs, 1)
	require.Contains(t, notif.msgs[0], "0.00019400")
	require.Contains(t, notif.msgs[0], "0.00020000")
	require.Contains(t, notif.msgs[0], "3.09%")
}
