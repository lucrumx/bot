package arbitragebot

import (
	"io"
	"testing"

	"github.com/lucrumx/bot/internal/exchange"
	"github.com/lucrumx/bot/internal/models"
	exchange_mocks "github.com/lucrumx/bot/internal/testmocks/exchange"
	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestBalanceStore_SetAndGetForAsset(t *testing.T) {
	store := newBalanceStore(zerolog.New(io.Discard))

	store.Set([]models.Balance{
		{
			ExchangeName: "ByBit",
			Asset: "BTCUSDT",
			Free: decimal.RequireFromString("10"),
			Locked: decimal.RequireFromString("1"),
			Total: decimal.RequireFromString("11"),
		},
	})

	got, ok := store.GetForAsset("ByBit", "BTCUSDT")
	assert.True(t, ok)
	assert.Equal(t, "ByBit", got.ExchangeName)
	assert.Equal(t, "BTCUSDT", got.Asset)
	assert.True(t, got.Total.Equal(decimal.RequireFromString("12")))
}

func TestBalanceStore_Get(t *testing.T) {
	bs := newBalanceStore(zerolog.New(io.Discard))

	bs.Set([]models.Balance{
		{
			ExchangeName: "ByBit",
			Asset: "BTCUSDT",
			Total: decimal.RequireFromString("10"),
		},
		{
			ExchangeName: "ByBit",
			Asset: "TONUSDT",
			Total: decimal.RequireFromString("20"),
		},
	})

	got, ok := bs.Get("ByBit")

	assert.True(t, ok)
	assert.Len(t, got, 2)

	assets := []string{got[0].Asset, got[1].Asset}
	assert.Equal(t, []string{"BTCUSDT", "TONUSDT"}, assets)
}

func TestBalanceStore_RetrieveBalances(t *testing.T) {
	store := newBalanceStore(zerolog.New(io.Discard))

	ctx := t.Context()

	okProvider := exchange_mocks.NewMockProvider(t)
	okProvider.EXPECT().GetBalances(ctx).Return([]models.Balance{
		{
			ExchangeName: "ByBit",
			Asset: "BTCUSDT",
			Total: decimal.RequireFromString("10"),
		},
	}, nil)


	store.retrieveBalances(ctx, []exchange.Provider{okProvider})

	got, ok := store.Get("ByBit")
	assert.True(t, ok)
	assert.Len(t, got, 1)
	assert.True(t, got[0].Total.Equal(decimal.RequireFromString("10")))
	assert.Equal(t, got[0].Asset, "BTCUSDT")
}
