package arbitragebot

import (
	"fmt"
	"io"
	"testing"

	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"github.com/lucrumx/bot/internal/exchange"
	"github.com/lucrumx/bot/internal/models"
	exchangeMocks "github.com/lucrumx/bot/internal/testmocks/exchange"
)

func TestBalanceStore_SetAndGetForAsset(t *testing.T) {
	store := newBalanceStore(zerolog.New(io.Discard))

	store.Set([]models.Balance{
		{
			ExchangeName: "ByBit",
			Asset:        "BTCUSDT",
			Free:         decimal.RequireFromString("10"),
			Locked:       decimal.RequireFromString("1"),
			Total:        decimal.RequireFromString("11"),
		},
	})

	got, ok := store.GetForAsset("ByBit", "BTCUSDT")
	fmt.Printf("%+v\n", got)
	assert.True(t, ok)
	assert.Equal(t, "ByBit", got.ExchangeName)
	assert.Equal(t, "BTCUSDT", got.Asset)
	assert.Equal(t, got.Total.String(), "11")
}

func TestBalanceStore_Get(t *testing.T) {
	bs := newBalanceStore(zerolog.New(io.Discard))

	bs.Set([]models.Balance{
		{
			ExchangeName: "ByBit",
			Asset:        "BTCUSDT",
			Total:        decimal.RequireFromString("10"),
		},
		{
			ExchangeName: "ByBit",
			Asset:        "TONUSDT",
			Total:        decimal.RequireFromString("20"),
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

	okProvider := exchangeMocks.NewMockProvider(t)
	okProvider.EXPECT().GetBalances(ctx).Return([]models.Balance{
		{
			ExchangeName: "ByBit",
			Asset:        "BTCUSDT",
			Total:        decimal.RequireFromString("10"),
		},
	}, nil)

	store.retrieveBalances(ctx, []exchange.Provider{okProvider})

	got, ok := store.Get("ByBit")
	assert.True(t, ok)
	assert.Len(t, got, 1)
	assert.True(t, got[0].Total.Equal(decimal.RequireFromString("10")))
	assert.Equal(t, got[0].Asset, "BTCUSDT")
}
