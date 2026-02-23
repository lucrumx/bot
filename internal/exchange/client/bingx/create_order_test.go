package bingx

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"github.com/lucrumx/bot/internal/config"
	"github.com/lucrumx/bot/internal/models"
)

func Test_CreateOrder_Integration(t *testing.T) {
	cfg := &config.Config{
		Exchange: config.ExchangeConfig{
			BingX: config.BingXConfig{
				APIKey:    "some-api-key",
				APISecret: "some-api-secret",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, cfg.Exchange.BingX.APIKey, r.Header.Get("X-BX-APIKEY"))

		query := r.URL.Query()
		assert.NotEmpty(t, query.Get("symbol"))
		assert.NotEmpty(t, query.Get("side"))
		assert.NotEmpty(t, query.Get("type"))
		assert.NotEmpty(t, query.Get("quantity"))
		assert.NotEmpty(t, query.Get("positionSide"))
		assert.NotEmpty(t, query.Get("clientOrderId"))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
            "code": 0,
            "msg": "",
            "data": {
                "order": {
                    "symbol": "BTC-USDT"
                }
            }
        }`))
	}))
	defer server.Close()

	ctx := t.Context()

	bingx := NewClient(cfg)
	bingx.baseURL = server.URL
	bingx.httpClient = server.Client()

	orderID, _ := uuid.NewV7()
	order := models.Order{
		ID:       orderID,
		Symbol:   "BTCUSDT",
		Side:     models.OrderSideBuy,
		Type:     models.OrderTypeMarket,
		Quantity: decimal.NewFromInt(1),
	}

	err := bingx.CreateOrder(ctx, order)

	assert.NoError(t, err)
}

func Test_CreateOrder_ValidateBeforeCreate(t *testing.T) {
	tests := []struct {
		name       string
		order      models.Order
		wantErrMsg string
	}{
		{
			name: "only market orders are supported",
			order: models.Order{
				Type: models.OrderTypeLimit,
			},
			wantErrMsg: "support only market orders",
		},
		{
			name: "quantity must be greater than zero",
			order: models.Order{
				Type:     models.OrderTypeMarket,
				Quantity: decimal.Zero,
			},
			wantErrMsg: "order quantity must be greater than 0",
		},
		{
			name: "symbol should be specified",
			order: models.Order{
				Type:     models.OrderTypeMarket,
				Quantity: decimal.NewFromInt(1),
			},
			wantErrMsg: "order symbol must be specified",
		},
		{
			name: "order ID should be uuid",
			order: models.Order{
				Type:     models.OrderTypeMarket,
				Quantity: decimal.NewFromInt(1),
				Symbol:   "BTCUSDT",
			},
			wantErrMsg: "order id must be valid uuid",
		},
		{
			name: "success",
			order: models.Order{
				Type:     models.OrderTypeMarket,
				Quantity: decimal.NewFromInt(1),
				Symbol:   "BTCUSDT",
				ID: func() uuid.UUID {
					id, _ := uuid.NewV7()
					return id
				}(),
			},
		},
	}

	for _, tt := range tests {
		err := validateBeforeCreateOrder(&tt.order)

		if tt.wantErrMsg != "" {
			assert.ErrorContains(t, err, tt.wantErrMsg, tt.name)
		} else {
			assert.NoError(t, err, tt.name)
		}
	}
}

func Test_CreateOrder_MapOrderToQuery(t *testing.T) {
	orderID := func() uuid.UUID {
		id, _ := uuid.NewV7()
		return id
	}()

	order := models.Order{
		Symbol:   "BTCUSDT",
		Side:     models.OrderSideBuy,
		Type:     models.OrderTypeMarket,
		Quantity: decimal.NewFromInt(1),
		ID:       orderID,
	}

	timestamp := time.Now().UnixMilli()

	res := mapRequestDataToOrderDTO(&order, timestamp)

	assert.Equal(t, map[string]string{
		"symbol":        "BTC-USDT",
		"side":          "BUY",
		"positionSide":  "LONG",
		"type":          "MARKET",
		"quantity":      "1",
		"clientOrderId": orderID.String(),
		"timestamp":     strconv.FormatInt(timestamp, 10),
	}, res)
}
