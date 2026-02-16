package bingx

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog/log"

	"github.com/lucrumx/bot/internal/config"
	"github.com/lucrumx/bot/internal/exchange"
)

// Metrics holds metrics related to websocket client operations.
type Metrics struct {
	droppedTrades atomic.Uint64
}

// wsClient represents a WebSocket client for BingX exchange.
type wsClient struct {
	Metrics *Metrics
	cfg     *config.Config
	wsMu    sync.Mutex
}

func newWsClient(cfg *config.Config) *wsClient {
	return &wsClient{
		Metrics: &Metrics{},
		cfg:     cfg,
	}
}

func (c *wsClient) writeJSON(wsConn *websocket.Conn, payload interface{}) error {
	c.wsMu.Lock()
	defer c.wsMu.Unlock()
	return wsConn.WriteJSON(payload)
}

func (c *wsClient) writeMessage(wsConn *websocket.Conn, messageType int, data []byte) error {
	c.wsMu.Lock()
	defer c.wsMu.Unlock()
	return wsConn.WriteMessage(messageType, data)
}

func (c *wsClient) writeControl(wsConn *websocket.Conn, messageType int, data []byte, deadline time.Time) error {
	c.wsMu.Lock()
	defer c.wsMu.Unlock()
	return wsConn.WriteControl(messageType, data, deadline)
}

func (c *wsClient) Start(ctx context.Context, symbols []string, outChan chan<- exchange.Trade) error {
	wsConn, _, err := websocket.DefaultDialer.Dial(c.cfg.Exchange.BingX.WSUrl, nil)
	if err != nil {
		return fmt.Errorf("failed to dial websocket: %w", err)
	}

	err = c.subscribe(wsConn, symbols)
	if err != nil {
		return fmt.Errorf("failed to subscribe to symbols: %w", err)
	}

	go c.readMessage(ctx, wsConn, outChan)
	go c.pingPongInterval(ctx, wsConn)
	go c.logMetrics(ctx)
	go func() {
		<-ctx.Done()
		_ = wsConn.Close()
	}()

	return nil
}

func (c *wsClient) subscribe(wsCon *websocket.Conn, symbols []string) error {
	for _, symbol := range symbols {
		id, err := uuid.NewV7()
		if err != nil {
			return fmt.Errorf("failed to generate uuid: %w", err)
		}

		dataType := fmt.Sprintf("%s@trade", strings.Replace(symbol, "USDT", "-USDT", 1))
		payload := map[string]string{
			"id":       id.String(),
			"reqType":  "sub",
			"dataType": dataType,
		}

		if err := c.writeJSON(wsCon, payload); err != nil {
			return fmt.Errorf("failed to send subscription request: %w", err)
		}
	}

	return nil
}

func (c *wsClient) readMessage(ctx context.Context, wsConn *websocket.Conn, outChan chan<- exchange.Trade) {
	defer func() {
		err := wsConn.Close()
		if err != nil && ctx.Err() == nil {
			log.Warn().Err(err).Msg("Failed to close websocket connection")
		}
	}()

	for {
		if ctx.Err() != nil {
			return
		}

		mt, messageByte, err := wsConn.ReadMessage()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Warn().Err(err).Msg("Failed to read message from BingX websocket")
			// TODO: something wrong with connection need to reconnect (reconnect not implemented yet)
			return
		}

		if mt == websocket.PingMessage {
			err = c.writeControl(wsConn, websocket.PongMessage, messageByte, time.Now().Add(time.Second*5))
			if err != nil {
				log.Warn().Err(err).Msg("Failed to send pong to BingX websocket")
			}
			continue
		}

		if mt == websocket.TextMessage {
			log.Warn().Msgf("text message? %s", string(messageByte))
		} else if mt == websocket.BinaryMessage {
			message, err := decodeGzip(messageByte)
			if err != nil {
				log.Warn().Err(err).Msg("Failed to decode gzip message from BingX websocket, trade sub")
			}

			if message == "Ping" {
				err = c.writeMessage(wsConn, websocket.TextMessage, []byte("Pong"))
				if err != nil {
					log.Warn().Err(err).Msg("Failed to send pong to BingX websocket (after decode binary)")
				}
				continue
			}

			if message == "Pong" {
				continue
			}

			var jsonMessage WsTradeMessageDTO
			if err := json.Unmarshal([]byte(message), &jsonMessage); err != nil {
				log.Warn().Err(err).Msgf("Failed to unmarshal message from BingX websocket message: %v", message)
			}

			if jsonMessage.Code != 0 {
				log.Warn().Msgf("error ws trade message, code not 0 %v", jsonMessage)
			}

			for _, val := range jsonMessage.Data {
				side := exchange.Buy
				if val.M {
					side = exchange.Sell
				}

				trade := exchange.Trade{
					Symbol: strings.TrimSuffix(val.Symbol, "-USDT") + "USDT",
					Ts:     val.T,
					Price:  float64(val.Price),
					Volume: float64(val.Volume),
					Side:   side,
				}

				select {
				case outChan <- trade:
				default:
					c.Metrics.droppedTrades.Add(1)
				}

			}
		}
	}
}

func decodeGzip(data []byte) (string, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	defer func() {
		_ = reader.Close()
	}()

	var decodedMsg []byte
	decodedMsg, err = io.ReadAll(reader)
	if err != nil {
		return "", err
	}

	return string(decodedMsg), nil
}

func (c *wsClient) pingPongInterval(ctx context.Context, wsConn *websocket.Conn) {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := c.writeMessage(wsConn, websocket.TextMessage, []byte("Ping"))
			if err != nil {
				log.Warn().Err(err).Msg("Failed to send ping to BingX websocket")
				return
			}
		}
	}
}

func (c *wsClient) logMetrics(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 30)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			droppedTradesCnt := c.Metrics.droppedTrades.Load()
			if droppedTradesCnt > 0 {
				log.Warn().Msgf("BingX metrics: dropped trades=%d", droppedTradesCnt)
			}
		}
	}
}
