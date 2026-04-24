package mexc

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"

	"github.com/lucrumx/bot/internal/config"
	"github.com/lucrumx/bot/internal/exchange"
	"github.com/lucrumx/bot/internal/exchange/client/mexc/dtos"
)

const wsPrivateOrderChannel = "push.personal.order"

// WsPrivateClient handles private WebSocket connections to MEXC exchange.
type WsPrivateClient struct {
	url    string
	cfg    *config.Config
	logger zerolog.Logger
	wsMut  sync.Mutex
	wsConn *websocket.Conn

	executionChannel    chan exchange.OrderExecutionEvent
	executionSubscribed bool
}

// NewWsPrivateClient initializes a WsPrivateClient.
func NewWsPrivateClient(cfg *config.Config, logger zerolog.Logger) *WsPrivateClient {
	return &WsPrivateClient{
		url:              cfg.Exchange.MEXC.WSUrl,
		cfg:              cfg,
		logger:           logger,
		executionChannel: make(chan exchange.OrderExecutionEvent, 100),
	}
}

// Start connects to the WebSocket, authenticates, and begins message processing.
func (c *WsPrivateClient) Start(ctx context.Context) error {
	wsConn, _, err := websocket.DefaultDialer.Dial(c.url, nil)
	if err != nil {
		return fmt.Errorf("MEXC ws private: failed to connect: %w", err)
	}

	c.wsConn = wsConn

	go func() {
		<-ctx.Done()
		_ = wsConn.Close()
	}()

	if err = c.login(); err != nil {
		return err
	}

	go c.pingPongInterval(ctx)

	go func() {
		defer c.closeChannels()

		for {
			select {
			case <-ctx.Done():
				return
			default:
				if err := c.handleMessage(); err != nil {
					c.logger.Error().Err(err).Str("exchange", "MEXC").Msg("error processing private ws message")
					_ = wsConn.Close()
					return
				}
			}
		}
	}()

	return nil
}

func (c *WsPrivateClient) login() error {
	reqTime := strconv.FormatInt(time.Now().UnixMilli(), 10)
	apiKey := c.cfg.Exchange.MEXC.APIKey

	h := hmac.New(sha256.New, []byte(c.cfg.Exchange.MEXC.APISecret))
	h.Write([]byte(apiKey + reqTime))
	signature := hex.EncodeToString(h.Sum(nil))

	payload := map[string]interface{}{
		"method":    "login",
		"subscribe": false,
		"param": map[string]string{
			"apiKey":    apiKey,
			"reqTime":   reqTime,
			"signature": signature,
		},
	}

	if err := c.writeJSON(payload); err != nil {
		return fmt.Errorf("MEXC ws private: failed to send login: %w", err)
	}

	_, raw, err := c.wsConn.ReadMessage()
	if err != nil {
		return fmt.Errorf("MEXC ws private: failed to read login response: %w", err)
	}

	var resp struct {
		Channel string `json:"channel"`
		Data    string `json:"data"`
	}
	if err := json.Unmarshal(raw, &resp); err != nil {
		return fmt.Errorf("MEXC ws private: failed to unmarshal login response: %w", err)
	}

	if resp.Channel != "rs.login" || resp.Data != "success" {
		return fmt.Errorf("MEXC ws private: login failed: %s", string(raw))
	}

	return nil
}

// SubscribeToExecutions subscribes to order state updates and returns the execution event channel.
func (c *WsPrivateClient) SubscribeToExecutions() (<-chan exchange.OrderExecutionEvent, error) {
	if !c.executionSubscribed {
		payload := map[string]interface{}{
			"method": "personal.filter",
			"param": map[string]interface{}{
				"filters": []map[string]string{
					{"filter": "order"},
				},
			},
		}

		if err := c.writeJSON(payload); err != nil {
			return nil, fmt.Errorf("MEXC ws private: failed to subscribe to order channel: %w", err)
		}

		c.executionSubscribed = true
	}

	return c.executionChannel, nil
}

func (c *WsPrivateClient) handleMessage() error {
	_, raw, err := c.wsConn.ReadMessage()
	if err != nil {
		return fmt.Errorf("MEXC ws private: failed to read message: %w", err)
	}

	var base dtos.WSBaseMessage
	if err := json.Unmarshal(raw, &base); err != nil {
		c.logger.Warn().Err(err).Msg("MEXC ws private: failed to unmarshal base message")
		return nil
	}

	if base.Channel != wsPrivateOrderChannel {
		return nil
	}

	var msg dtos.WSOrderDTO
	if err := json.Unmarshal(raw, &msg); err != nil {
		c.logger.Warn().Err(err).Msg("MEXC ws private: failed to unmarshal order message")
		return nil
	}

	d := msg.Data

	if d.State == dtos.OrderStateInvalid {
		c.logger.Warn().
			Str("symbol", d.Symbol).
			Str("orderId", d.OrderID).
			Int("errorCode", d.ErrorCode).
			Msg("MEXC ws private: order invalid")
		return nil
	}

	if d.State != dtos.OrderStateFilled {
		return nil
	}

	orderID, err := parseUUIDFromHex(d.ExternalOid)
	if err != nil {
		c.logger.Warn().Err(err).Str("externalOid", d.ExternalOid).Msg("MEXC ws private: failed to parse externalOid as UUID")
		return nil
	}

	c.executionChannel <- exchange.OrderExecutionEvent{
		OrderID:    orderID,
		ExecPrice:  d.DealAvgPrice,
		ExecQty:    d.DealVol,
		ExecValue:  d.DealAvgPrice.Mul(d.DealVol),
		LeavesQty:  d.RemainVol,
		OrderPrice: d.Price,
		OrderQty:   d.Vol,
	}

	return nil
}

// parseUUIDFromHex parses a 32-char hex UUID string (no dashes) into uuid.UUID.
func parseUUIDFromHex(s string) (uuid.UUID, error) {
	if len(s) == 32 {
		s = s[:8] + "-" + s[8:12] + "-" + s[12:16] + "-" + s[16:20] + "-" + s[20:]
	}
	return uuid.Parse(s)
}

func (c *WsPrivateClient) pingPongInterval(ctx context.Context) {
	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := c.writeJSON(map[string]string{"method": "ping"}); err != nil {
				c.logger.Warn().Err(err).Msg("MEXC ws private: failed to send ping")
				return
			}
		}
	}
}

func (c *WsPrivateClient) writeJSON(payload interface{}) error {
	c.wsMut.Lock()
	defer c.wsMut.Unlock()
	return c.wsConn.WriteJSON(payload)
}

func (c *WsPrivateClient) closeChannels() {
	close(c.executionChannel)
}
