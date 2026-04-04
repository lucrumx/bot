package bingx

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lucrumx/bot/internal/config"
	"github.com/lucrumx/bot/internal/exchange/client/bingx/dtos"
)

var testUpgrader = websocket.Upgrader{
	CheckOrigin: func(_ *http.Request) bool { return true },
}

func gzipEncode(t *testing.T, data string) []byte {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, err := gz.Write([]byte(data))
	require.NoError(t, err)
	require.NoError(t, gz.Close())
	return buf.Bytes()
}

func TestWsPrivateClient_Happy(t *testing.T) {
	httpSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/openApi/user/auth/userDataStream", r.URL.Path)

		// no need to check signature, it's tested in the prepare_query_test.go'
		assert.NotEmpty(t, r.URL.Query().Get("signature"))
		assert.NotEmpty(t, r.URL.Query().Get("timestamp"))

		resp := dtos.GenerateListenKeyDTO{
			Code: 0,
			Msg:  "success",
		}
		resp.Data.ListenKey = "some-listen-key-123"

		w.Header().Set("Content-Type", "application/json")
		b, _ := json.Marshal(resp)
		_, _ = w.Write(b)

	}))

	wsServ := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "some-listen-key-123", r.URL.Query().Get("listenKey"))

		conn, err := testUpgrader.Upgrade(w, r, nil)
		require.NoError(t, err)
		defer func() {
			_ = conn.Close()
		}()

		execDTO := map[string]interface{}{
			"s":  "BTCUSDT",
			"c":  "550e8400-e29b-41d4-a716-446655440000", // client order ID (uuid)
			"i":  123456789,                              // BingX order ID
			"S":  "BUY",
			"o":  "MARKET",
			"q":  "0.01",    // order qty
			"p":  "0",       // order price (market = 0)
			"ap": "50000.5", // avg filled price
			"z":  "0.01",    // filled qty (= qty → fully filled)
			"X":  "FILLED",
			"tv": "500.005", // trade value
			"n":  "-0.05",   // fee
			"N":  "USDT",
		}

		privateMsg := map[string]interface{}{
			"e": "TRADE_UPDATE",
			"E": time.Now().UnixMilli(),
			"o": execDTO,
		}

		payload, err := json.Marshal(privateMsg)
		require.NoError(t, err)

		compressed := gzipEncode(t, string(payload))
		err = conn.WriteMessage(websocket.BinaryMessage, compressed)
		require.NoError(t, err)

		// open while client is subscribed and not close connection
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}

	}))

	defer httpSrv.Close()
	defer wsServ.Close()

	wsURL := "ws" + strings.TrimPrefix(wsServ.URL, "http")

	cfg := &config.Config{
		Exchange: config.ExchangeConfig{
			BingX: config.BingXConfig{
				APIBaseURL:       httpSrv.URL,
				WSPrivateSwapURL: wsURL,
				APIKey:           "test-api-key",
				APISecret:        "test-api-secret",
			},
		},
	}

	logger := zerolog.Nop()
	client := NewWsPrivateClient(cfg, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.Start(ctx)
	require.NoError(t, err)

	execCh, err := client.SubscribeToExecutions()
	require.NoError(t, err)

	select {
	case event := <-execCh:
		assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", event.OrderID.String())
		assert.True(t, event.ExecPrice.StringFixed(1) == "50000.5") // ap → ExecPrice
		assert.True(t, event.ExecQty.StringFixed(2) == "0.01")      // z → ExecQty
		assert.True(t, event.LeavesQty.IsZero())                    // q - z = 0
		assert.True(t, event.ExecValue.StringFixed(3) == "500.005") // tv → ExecValue
	case <-time.After(3 * time.Second):
		t.Fatal("timeout waiting for execution event")
	}
}
