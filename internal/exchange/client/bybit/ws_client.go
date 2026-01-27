package bybit

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"

	"github.com/lucrumx/bot/internal/utils"

	"github.com/lucrumx/bot/internal/exchange"
)

const linearPublicWsURL = "/v5/public/linear"
const batchSize = 20
const pingPongInterval = 20

// Metrics represents metrics for the WebSocket client.
type Metrics struct {
	DroppedTrades atomic.Uint64
}

// wsClient represents a WebSocket client for ByBit exchange.
type wsClient struct {
	url     string
	Metrics *Metrics
}

func newWsClient() *wsClient {
	baseURL := utils.GetEnv("BYBIT_WS_BASE_URL", "")
	return &wsClient{
		url:     baseURL + linearPublicWsURL,
		Metrics: &Metrics{},
	}
}

func (c *wsClient) Start(ctx context.Context, symbols []string, outChan chan<- exchange.Trade) error {
	wsConn, _, err := websocket.DefaultDialer.Dial(c.url, nil)
	if err != nil {
		return fmt.Errorf("failed to dial websocket: %w", err)
	}

	if err := c.subscribeBatch(wsConn, symbols); err != nil {
		_ = wsConn.Close()
		return fmt.Errorf("failed to subscribe to symbols: %w", err)
	}

	go c.pingPong(ctx, wsConn)
	go c.readMessages(ctx, wsConn, outChan)
	go c.LogMetric(ctx)
	go func() {
		<-ctx.Done()
		_ = wsConn.Close()
	}()

	return nil
}

// subscribeButch sends subscription requests for a batch of symbols to the WebSocket connection.
func (c *wsClient) subscribeBatch(wsConn *websocket.Conn, symbols []string) error {
	for i := 0; i < len(symbols); i += batchSize {
		end := i + batchSize
		if end > len(symbols) {
			end = len(symbols)
		}

		batch := symbols[i:end]
		args := make([]string, len(batch))
		for j, symbol := range batch {
			args[j] = fmt.Sprintf("publicTrade.%s", symbol)
		}

		subReq := map[string]interface{}{
			"op":     "subscribe",
			"req_id": fmt.Sprintf("sub-%d-%d", time.Now().UnixNano(), i),
			"args":   args,
		}

		if err := wsConn.WriteJSON(subReq); err != nil {
			return fmt.Errorf("failed to send subscription request: %w", err)
		}
	}

	return nil
}

func (c *wsClient) pingPong(ctx context.Context, wsConn *websocket.Conn) {
	ticker := time.NewTicker(pingPongInterval * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pinPongPayload := map[string]interface{}{
				"op": "ping",
			}
			if err := wsConn.WriteJSON(pinPongPayload); err != nil {
				log.Printf("failed to send pin pong: %v", err)
				return
			}
		}
	}
}

func (c *wsClient) readMessages(ctx context.Context, wsConn *websocket.Conn, outChan chan<- exchange.Trade) {
	defer func() {
		err := wsConn.Close()
		if err != nil && ctx.Err() == nil { // log only if context did not close the connection (context still alive)
			log.Printf("failed to close websocket connection: %v", err)
		}
	}()

	for {
		if ctx.Err() != nil {
			return
		}

		_, messageByte, err := wsConn.ReadMessage()
		if err != nil {
			if ctx.Err() != nil {
				// Not ReadMessage error, connection already closed
				return
			}

			// TODO: тут по идее реконект
			log.Printf("failed to read message: %v", err)
			return
		}

		var message wsTradeMessageDTO
		if err := json.Unmarshal(messageByte, &message); err != nil {
			log.Printf("failed to unmarshal message: %v", err)
			continue
		}

		if message.Topic != "" {
			for _, t := range message.Data {
				trade, err := mapWsTrade(t)
				if err != nil {
					log.Printf("failed to map trade: %v", err)
				}

				select {
				case outChan <- trade:
				default:
					log.Printf("chain is full, dropped trade: %v", trade)
					c.Metrics.DroppedTrades.Add(1)
				}
			}
		}
	}
}

// LogMetric logs metrics related to the websocket client.
func (c *wsClient) LogMetric(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			log.Printf("metrics: dropped trades=%d", c.Metrics.DroppedTrades.Load())
		case <-ctx.Done():
			return
		}
	}
}
