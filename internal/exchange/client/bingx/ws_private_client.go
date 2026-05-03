package bingx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"

	"github.com/lucrumx/bot/internal/config"
	"github.com/lucrumx/bot/internal/exchange"
	"github.com/lucrumx/bot/internal/exchange/client/bingx/dtos"
	wstopics "github.com/lucrumx/bot/internal/exchange/client/bingx/ws_topics"
)

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
		url:    cfg.Exchange.BingX.WSPrivateSwapURL,
		cfg:    cfg,
		logger: logger,

		executionChannel:    make(chan exchange.OrderExecutionEvent, 100),
		executionSubscribed: false,
	}
}

// Start initializes the private webSocket connection, performs authentication, and starts message handling and ping routines.
func (c *WsPrivateClient) Start(ctx context.Context) error {
	if c.executionSubscribed {
		return nil
	}

	listenKey, err := c.getListenKey(ctx)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s?listenKey=%s", c.url, listenKey)

	wsConn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return fmt.Errorf("BingX ws private: failed to connect to websocket: %w", err)
	}

	c.wsConn = wsConn

	go func() {
		<-ctx.Done()
		_ = wsConn.Close()
	}()

	go c.pingPongInterval(ctx)

	go func() {
		defer c.closeChannels()

		for {
			select {
			case <-ctx.Done():
				return
			default:
				err := c.handleMessage()
				if err != nil {
					c.logger.Error().Err(err).Str("exchange", "BingX").Msg("error processing message")
					_ = wsConn.Close()
					return
				}
			}
		}
	}()

	c.executionSubscribed = true

	return nil
}

func (c *WsPrivateClient) pingPongInterval(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 5)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			err := c.writeMessage([]byte("Ping"))
			if err != nil {
				c.logger.Warn().Err(err).Msg("Failed to send ping to BingX websocket")
				return
			}
		}
	}
}

// SubscribeToExecutions returns the execution events channel.
// BingX listenKey-based connections automatically receive all account events.
func (c *WsPrivateClient) SubscribeToExecutions() (<-chan exchange.OrderExecutionEvent, error) {
	return c.executionChannel, nil
}

func (c *WsPrivateClient) handleMessage() error {
	mt, raw, err := c.wsConn.ReadMessage()

	if err != nil {
		return fmt.Errorf("BingX ws private: failed to read message from. Network issue? %v", err)
	}

	if mt == websocket.PingMessage {
		err = c.writeControl(websocket.PongMessage, []byte("Pong"), time.Now().Add(time.Second*5))
		if err != nil {
			c.logger.Warn().Err(err).Msg("Failed to send pong to BingX websocket")
		}
		return nil
	}

	if mt == websocket.TextMessage {
		c.logger.Warn().Msgf("BingX ws private: text message: %s", string(raw))
		return nil
	}

	if mt != websocket.BinaryMessage {
		c.logger.Warn().Str("exchange", "BingX").Msg("unexpected message type")
		return nil
	}

	message, err := decodeGzip(raw)
	if err != nil {
		c.logger.Warn().Err(err).Str("exchange", "BingX").Msg("Failed to decode gzip message from BingX private websocket")
		return nil
	}

	if message == "Ping" || message == "Pong" {
		if message == "Ping" {
			err = c.writeMessage([]byte("Pong"))
			if err != nil {
				c.logger.Warn().Err(err).Str("exchange", "BingX").Msg("Failed to send pong to BingX websocket (after decode binary)")
			}
		}
		return nil
	}

	var r dtos.PrivateMessageDTO
	if err := json.Unmarshal([]byte(message), &r); err != nil {
		c.logger.Warn().Err(err).Str("exchange", "BingX").Msg("BingX ws private: failed to unmarshal response message")
		return nil
	}

	switch r.EventType {
	case wstopics.PrivateExecutionEvent:
		order, ok := c.handleExecutionEvent(&r)
		if !ok { // if error when unmarshaling
			return nil
		}
		// Blocking, but channel has buffer
		c.executionChannel <- order
	default:
		c.logger.Warn().Str("event_type", r.EventType).Msg("BingX ws private: unknown event type")
	}

	return nil
}

func (c *WsPrivateClient) handleExecutionEvent(message *dtos.PrivateMessageDTO) (exchange.OrderExecutionEvent, bool) {
	var execution dtos.ExecutionDTO
	if err := json.Unmarshal(message.O, &execution); err != nil {
		c.logger.Warn().Err(err).Msg("BingX ws private: failed to unmarshal execution message from BingX privates websocket")
		return exchange.OrderExecutionEvent{}, false
	}

	return exchange.OrderExecutionEvent{
		OrderID:         execution.OrderID,
		ExchangeOrderID: strconv.FormatInt(execution.BingXOrderID, 10),
		ExecPrice:       execution.AvgPrice,
		ExecQty:         execution.FilledQty,
		ExecValue:       execution.TradeValue,
		LeavesQty:       execution.Qty.Sub(execution.FilledQty),
		OrderPrice:      execution.Price,
		OrderQty:        execution.Qty,
	}, true
}

func (c *WsPrivateClient) writeMessage(data []byte) error {
	c.wsMut.Lock()
	defer c.wsMut.Unlock()
	return c.wsConn.WriteMessage(websocket.TextMessage, data)
}

func (c *WsPrivateClient) writeControl(messageType int, data []byte, deadline time.Time) error {
	c.wsMut.Lock()
	defer c.wsMut.Unlock()
	return c.wsConn.WriteControl(messageType, data, deadline)
}

func (c *WsPrivateClient) closeChannels() {
	close(c.executionChannel)
}

func (c *WsPrivateClient) getListenKey(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.cfg.Exchange.BingX.APIBaseURL+"/openApi/user/auth/userDataStream", nil)
	if err != nil {
		return "", fmt.Errorf("BingX client failed to create request (ws private client): %w", err)
	}

	query := make(map[string]string)
	queryStr := getSortedQuery(query, time.Now().UnixMilli(), false)
	signature := computeHmac256(c.cfg, queryStr)
	req.URL.RawQuery = fmt.Sprintf("%s&signature=%s", getSortedQuery(query, 0, true), signature)

	req.Header.Set("X-BX-APIKEY", c.cfg.Exchange.BingX.APIKey)

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("BingX client http get tickers request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		fmt.Printf("err %v\n", err)
		return "", fmt.Errorf("BingX client unexpected http while getting listen key status code: %d, %s", resp.StatusCode, body)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("BingX client failed to read get listen key response body: %w", err)
	}

	var r dtos.GenerateListenKeyDTO
	if err := json.Unmarshal(body, &r); err != nil {
		return "", fmt.Errorf("BingX client failed to unmarshal get listen key response: %w", err)
	}

	if r.Code != 0 {
		return "", fmt.Errorf("BingX client failed to get listen key, code is not 0: %v", r)
	}

	return r.Data.ListenKey, nil
}
