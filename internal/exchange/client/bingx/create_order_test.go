package bingx

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/lucrumx/bot/internal/exchange/client/bingx/dtos"

	"github.com/lucrumx/bot/internal/config"
	"github.com/lucrumx/bot/internal/models"
)

func Test_CreateOrder(t *testing.T) {
	cfg := &config.Config{
		Exchange: config.ExchangeConfig{
			BingX: config.BingXConfig{
				// APIKey:    "some-api-key",
				APIKey: "",
				//APISecret: "some-api-secret",
				APISecret: "",
			},
		},
	}

	bingx := NewClient(cfg)

	ctx := t.Context()

	cid, _ := uuid.NewV7()
	order := dtos.OrderCreateRequestDTO{
		Symbol:        "BTC-USDT",
		Side:          models.OrderSideBuy,
		Type:          models.OrderTypeMarket,
		Quantity:      1,
		ClientOrderID: cid,
	}

	fmt.Printf("%+v\n", order)

	res, err := bingx.CreateOrder(ctx, order)

	assert.NoError(t, err)
	assert.NotNil(t, &res)

	fmt.Printf("%+v\n", res)
}
