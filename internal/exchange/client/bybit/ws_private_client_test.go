package bybit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lucrumx/bot/internal/config"
)

const orderID = "e48d152d-1d81-479b-ba4a-94d6259afd00"

var testUpgrader = websocket.Upgrader{
	CheckOrigin: func(_ *http.Request) bool { return true },
}

func getConfig(wsURL string) *config.Config {
	return &config.Config{
		Exchange: config.ExchangeConfig{
			ByBit: config.ByBitConfig{
				WsBaseURL: wsURL, // тестовый сервер вместо реального
				APIKey:    "test-api-key",
				APISecret: "test-api-secret",
			},
		},
	}
}

func getServer(t *testing.T, falseAuth bool) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := testUpgrader.Upgrade(w, r, nil)
		assert.NoError(t, err)
		defer func() {
			_ = conn.Close()
		}()

		// every ReadMessage block executions and test do it step by step

		// read auth request
		_, msg, err := conn.ReadMessage()
		assert.NoError(t, err)
		var authReq map[string]interface{}
		assert.NoError(t, json.Unmarshal(msg, &authReq))
		assert.Equal(t, "auth", authReq["op"])

		// Auth response
		authResp := map[string]interface{}{
			"success": !falseAuth,
		}
		require.NoError(t, conn.WriteJSON(authResp))

		// Read subscribe request
		_, msg, err = conn.ReadMessage()
		require.NoError(t, err)

		var subReq struct {
			Op   string   `json:"op"`
			Args []string `json:"args"`
		}
		assert.NoError(t, json.Unmarshal(msg, &subReq))
		assert.Equal(t, "subscribe", subReq.Op)
		assert.Equal(t, 1, len(subReq.Args))
		assert.Equal(t, subReq.Args[0], "execution")

		// Send execution event
		execMsg := map[string]interface{}{
			"topic":        "execution",
			"id":           "test-id-1",
			"creationTime": time.Now().UnixMilli(),
			"data": []map[string]interface{}{
				{
					"symbol":      "BTCUSDT",
					"execPrice":   "50000.5",
					"execQty":     "0.001",
					"execValue":   "50.0005",
					"leavesQty":   "0",
					"orderID":     "exchange-order-123",
					"orderLinkId": orderID,
					"orderPrice":  "50000",
					"orderQty":    "0.001",
					"side":        "Buy",
					"execType":    "Trade",
					"category":    "linear",
				},
			},
		}
		assert.NoError(t, conn.WriteJSON(execMsg))

		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}))
}

// Happy auth and subscribe to the execution topic
func TestWsPrivateClient_Happy(t *testing.T) {
	// --- Mock WS Server ---
	srv := getServer(t, false)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	cfg := getConfig(wsURL)

	logger := zerolog.Nop()
	client := NewWsPrivateClient(cfg, logger)

	client.url = wsURL

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start and Subscribe
	err := client.Start(ctx)
	require.NoError(t, err)

	execCh, err := client.SubscribeToExecutions()
	require.NoError(t, err)

	// --- Проверяем execution event ---
	select {
	case event := <-execCh:
		assert.Equal(t, orderID, event.OrderID.String())
		assert.True(t, event.ExecPrice.Equal(decimal.RequireFromString("50000.5")))
		assert.True(t, event.ExecQty.Equal(decimal.RequireFromString("0.001")))
		assert.True(t, event.LeavesQty.Equal(decimal.Zero))
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for execution event")
	}
}

// Fail auth in ws private
func TestWsPrivateClient_AuthFailure(t *testing.T) {
	// --- Mock WS Server ---
	srv := getServer(t, true)
	defer srv.Close()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")

	cfg := getConfig(wsURL)

	logger := zerolog.Nop()
	client := NewWsPrivateClient(cfg, logger)

	client.url = wsURL

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Start and Subscribe
	err := client.Start(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "auth response not successful")
}
