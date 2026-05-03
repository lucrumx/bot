package bybit

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"

	"github.com/lucrumx/bot/internal/config"
	"github.com/lucrumx/bot/internal/exchange"
	"github.com/lucrumx/bot/internal/exchange/client/bybit/dtos"
	wstopics "github.com/lucrumx/bot/internal/exchange/client/bybit/ws_topics"
)

const wsPrivateURL = "/v5/private"

// WsPrivateClient handles private WebSocket connections to the exchange, including authentication and message processing.
type WsPrivateClient struct {
	url    string
	cfg    *config.Config
	logger zerolog.Logger
	wsMut  sync.Mutex
	wsConn *websocket.Conn

	executionChannel    chan exchange.OrderExecutionEvent
	executionSubscribed bool
}

// NewWsPrivateClient initializes a WsPrivateClient with the given configuration and logger for private WebSocket connections.
func NewWsPrivateClient(cfg *config.Config, logger zerolog.Logger) *WsPrivateClient {
	return &WsPrivateClient{
		url:    cfg.Exchange.ByBit.WsBaseURL + wsPrivateURL,
		cfg:    cfg,
		logger: logger,

		executionChannel:    make(chan exchange.OrderExecutionEvent, 100),
		executionSubscribed: false,
	}
}

// Start initializes the private webSocket connection, performs authentication, and starts message handling and ping routines.
func (c *WsPrivateClient) Start(ctx context.Context) error {
	wsConn, _, err := websocket.DefaultDialer.Dial(c.url, nil)
	if err != nil {
		return fmt.Errorf("ByBit ws private: failed to connect to websocket: %w", err)
	}

	c.wsConn = wsConn

	go func() {
		<-ctx.Done()
		_ = wsConn.Close()
	}()

	if err = c.auth(); err != nil {
		return err
	}

	go c.pingPing(ctx)

	go func() {
		defer c.closeChannels()

		for {
			select {
			case <-ctx.Done():
				return
			default:
				err := c.handleMessage()
				if err != nil {
					c.logger.Error().Err(err).Str("exchange", "bybit").Msg("error processing message")
					_ = wsConn.Close()
					return
				}
			}
		}
	}()

	return nil
}

func (c *WsPrivateClient) auth() error {
	// timestamp
	expires := time.Now().UnixMilli() + 5000

	// Auth
	apiKey := c.cfg.Exchange.ByBit.APIKey
	strForSign := fmt.Sprintf("GET/realtime%d", expires)
	signature := sign(c.cfg.Exchange.ByBit.APISecret, strForSign)
	authReq := map[string]interface{}{
		"op": "auth",
		"args": []interface{}{
			apiKey,
			expires,
			signature,
		},
	}

	if err := c.writeJSON(authReq); err != nil {
		return fmt.Errorf("BiBit ws private: failed to send auth request: %w", err)
	}

	// read the first message on auth
	mt, raw, err := c.wsConn.ReadMessage()
	if err != nil {
		return fmt.Errorf("BiBit ws private: failed to read first websocket message: %w", err)
	}

	if mt != websocket.TextMessage {
		return fmt.Errorf("BiBit ws private: expected text message, got %d", mt)
	}

	var authMessage dtos.AuthRespDTO
	if err := json.Unmarshal(raw, &authMessage); err != nil {
		return fmt.Errorf("BiBit ws private: failed to unmarshal auth response: %w", err)
	}

	if !authMessage.Success {
		return fmt.Errorf("BiBit ws private: auth response not successful, %v", authMessage)
	}

	return nil
}

func (c *WsPrivateClient) pingPing(ctx context.Context) {
	ticker := time.NewTicker(pingPongInterval * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pinPongMessage := map[string]interface{}{
				"op": "ping",
			}
			if err := c.writeJSON(pinPongMessage); err != nil {
				c.logger.Warn().Err(err).Msg("BiBit ws private: failed to send ping pong")
			}
		}
	}
}

// SubscribeToExecutions subscribe to execution stream
func (c *WsPrivateClient) SubscribeToExecutions() (<-chan exchange.OrderExecutionEvent, error) {
	if !c.executionSubscribed {
		payload := map[string]interface{}{
			"op":   "subscribe",
			"args": [1]string{"execution"},
		}

		if err := c.writeJSON(payload); err != nil {
			return nil, fmt.Errorf("BiBit ws private: failed to subscribe to execution stream: %w", err)
		}

		mu := sync.Mutex{}
		mu.Lock()
		c.executionSubscribed = true
		mu.Unlock()
	}

	return c.executionChannel, nil
}

func (c *WsPrivateClient) handleMessage() error {
	mt, raw, err := c.wsConn.ReadMessage()

	if err != nil {
		return fmt.Errorf("BiBit ws private: failed to read message from. Network issue? %v", err)
	}

	if mt != websocket.TextMessage {
		return nil
	}

	var message dtos.MessageDTO
	if err := json.Unmarshal(raw, &message); err != nil {
		c.logger.Warn().Err(err).Msg("BiBit ws private: failed to unmarshal response message")
		return nil
	}

	switch message.Topic {
	case wstopics.Execution:
		orders := c.handleExecutionEvent(&message)
		if len(orders) > 0 {
			for _, order := range orders {
				// Blocking, but channel has buffer
				c.executionChannel <- order
			}
		}
	}

	return nil
}

func (c *WsPrivateClient) handleExecutionEvent(message *dtos.MessageDTO) []exchange.OrderExecutionEvent {
	var executions []dtos.ExecutionDTO
	if err := json.Unmarshal(message.Data, &executions); err != nil {
		c.logger.Warn().Err(err).Msg("BiBit ws private: failed to unmarshal execution message from ByBit privates websocket")
		return nil
	}

	orders := map[uuid.UUID]exchange.OrderExecutionEvent{}
	for _, execution := range executions {
		orderID := execution.OrderLinkID

		if _, ok := orders[orderID]; !ok {
			orders[orderID] = exchange.OrderExecutionEvent{
				OrderID:         execution.OrderLinkID,
				ExchangeOrderID: execution.OrderID,
				ExecPrice:       execution.ExecPrice,
				ExecQty:         execution.ExecQty,
				ExecValue:       execution.ExecValue,
				LeavesQty:       execution.LeavesQty,
				OrderPrice:      execution.OrderPrice,
				OrderQty:        execution.OrderQty,
			}
		} else {
			order := orders[orderID]
			order.ExecQty = order.ExecQty.Add(execution.ExecQty)
			order.ExecValue = order.ExecValue.Add(execution.ExecValue)
			order.ExecPrice = order.ExecValue.Div(order.ExecQty)
			order.LeavesQty = execution.LeavesQty
			orders[orderID] = order
		}
	}

	res := make([]exchange.OrderExecutionEvent, 0, len(orders))
	for _, order := range orders {
		res = append(res, order)
	}

	return res
}

func (c *WsPrivateClient) writeJSON(payload interface{}) error {
	c.wsMut.Lock()
	defer c.wsMut.Unlock()
	return c.wsConn.WriteJSON(payload)
}

func (c *WsPrivateClient) closeChannels() {
	close(c.executionChannel)
}
