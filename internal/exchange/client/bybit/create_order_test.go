package bybit

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"

	"github.com/lucrumx/bot/internal/exchange/client/bybit/dtos"

	"github.com/lucrumx/bot/internal/config"
	"github.com/lucrumx/bot/internal/models"
)

func Test_CreateOrder_Integration(t *testing.T) {
	cfg := &config.Config{
		Exchange: config.ExchangeConfig{
			ByBit: config.ByBitConfig{
				BaseURL:    "http://localhost:8080",
				APIKey:     "some-api-key",
				APISecret:  "some-api-secret",
				RecvWindow: 5000,
			},
		},
	}

	var bodyBytes []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, cfg.Exchange.ByBit.APIKey, r.Header.Get("X-BAPI-API-KEY"))
		hTimestamp, err := strconv.ParseInt(r.Header.Get("X-BAPI-TIMESTAMP"), 10, 64)
		assert.NoError(t, err)
		assert.IsType(t, int64(0), hTimestamp)
		recvWindow := r.Header.Get("X-BAPI-RECV-WINDOW")
		assert.Equal(t, "5000", recvWindow)

		bodyBytes, err = io.ReadAll(r.Body)
		assert.NoError(t, err)
		bodyString := string(bodyBytes)
		fmt.Printf("Body bodyString: %s\n", bodyString)
		tStr := strconv.FormatInt(hTimestamp, 10)
		signStr := tStr + cfg.Exchange.ByBit.APIKey + recvWindow + bodyString
		fmt.Printf("Sign string test: %s\n", signStr)

		h := hmac.New(sha256.New, []byte(cfg.Exchange.ByBit.APISecret))
		h.Write([]byte(signStr))
		signature := hex.EncodeToString(h.Sum(nil))

		assert.Equal(t, signature, r.Header.Get("X-BAPI-SIGN"))

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"retCode": 0,
			"retMsg": "OK",
			"result": {
				"orderId": "1321003749386327552",
				"orderLinkId": "spot-test-postonly"
			},
			"retExtInfo": {},
			"time": 1672211918471
		}`))
	}))
	defer server.Close()

	ctx := t.Context()

	bybit := NewByBitClient(cfg)
	bybit.baseURL = server.URL
	bybit.http = server.Client()

	orderID, _ := uuid.NewV7()
	order := models.Order{
		ID:       orderID,
		Symbol:   "BTCUSDT",
		Side:     models.OrderSideBuy,
		Type:     models.OrderTypeMarket,
		Market:   models.OrderMarketLinear,
		Quantity: decimal.NewFromInt(1),
	}

	err := bybit.CreateOrder(ctx, order)

	assert.NoError(t, err)

	var payload map[string]interface{}
	err = json.Unmarshal(bodyBytes, &payload)
	assert.NoError(t, err)

	category, ok := payload["category"].(string)
	assert.True(t, ok)
	assert.Equal(t, "linear", category)

	symbol, ok := payload["symbol"].(string)
	assert.True(t, ok)
	assert.Equal(t, order.Symbol, symbol)

	side, ok := payload["side"].(string)
	assert.True(t, ok)
	assert.Equal(t, side, string(dtos.OrderSideBuy))

	orderType, ok := payload["orderType"].(string)
	assert.True(t, ok)
	assert.Equal(t, orderType, string(dtos.OrderTypeMarket))

	qty, ok := payload["qty"].(string)
	assert.True(t, ok)
	assert.Equal(t, qty, order.Quantity.String())

	id, ok := payload["orderLinkId"].(string)
	fmt.Printf("Order payload: %v\n", payload)
	assert.True(t, ok)
	assert.Equal(t, id, order.ID.String())
}
