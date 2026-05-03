package arbitragebot

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/lucrumx/bot/internal/config"
	"github.com/lucrumx/bot/internal/models"
	"github.com/lucrumx/bot/internal/utils"
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
func (r *repoStub) FindOne(_ context.Context, _ FindFilter) (*models.ArbitrageSpread, error) {
	return nil, nil
}
func (r *repoStub) FindOneByOrderID(_ context.Context, _ uuid.UUID) (*models.ArbitrageSpread, error) {
	return &models.ArbitrageSpread{}, nil
}

func TestFormatPrice_PreservesLowPricePrecision(t *testing.T) {
	require.Equal(t, "0.00019400", utils.FormatPrice(0.000194))
	require.Equal(t, "0.00020000", utils.FormatPrice(0.0002))
	require.Equal(t, "1.2345", utils.FormatPrice(1.2345))
}

func TestEngine_HandleOpen_ShowsDistinctLowPrices(t *testing.T) {
	notif := &notifierStub{}
	engine := NewEngine(&config.Config{}, nil, nil, &repoStub{}, notif, zerolog.Nop())

	engine.handleOpen(context.Background(), &SpreadEvent{
		Status:            models.ArbitrageSpreadOpened,
		Symbol:            "AKEUSDT",
		BuyOnExchange:     "ByBit",
		SellOnExchange:    "BingX",
		BuyPrice:          0.000194,
		SellPrice:         0.0002,
		FromSpreadPercent: 3.09,
		MaxSpreadPercent:  3.09,
	})

	time.Sleep(50 * time.Millisecond) // wait for notification goroutine

	require.Len(t, notif.msgs, 1)
	require.Contains(t, notif.msgs[0], "0.00019400")
	require.Contains(t, notif.msgs[0], "0.00020000")
	require.Contains(t, notif.msgs[0], "3.09%")
}
