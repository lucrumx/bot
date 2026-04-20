package mexc

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"

	"github.com/lucrumx/bot/internal/config"
	"github.com/lucrumx/bot/internal/exchange"
	"github.com/lucrumx/bot/internal/exchange/client/mexc/dtos"
)

// Metrics holds metrics related to websocket client operations.
type Metrics struct {
	droppedTrades atomic.Uint64
}

// wsClient represents a WebSocket client for MEXC exchange.
type wsClient struct {
	Metrics *Metrics
	cfg     *config.Config
	wsMu    sync.Mutex
	logger  zerolog.Logger
}

func newWsClient(cfg *config.Config, logger zerolog.Logger) *wsClient {
	return &wsClient{
		Metrics: &Metrics{},
		cfg:     cfg,
		logger:  logger,
	}
}

func (c *wsClient) writeJSON(wsConn *websocket.Conn, payload interface{}) error {
	c.wsMu.Lock()
	defer c.wsMu.Unlock()
	return wsConn.WriteJSON(payload)
}

func (c *wsClient) Start(ctx context.Context, symbols []string, category exchange.Category, outChan chan<- exchange.Trade) error {
	if category != exchange.CategoryLinear {
		return fmt.Errorf("mexc websocket supports only %s trades", exchange.CategoryLinear)
	}

	wsConn, _, err := websocket.DefaultDialer.Dial(c.cfg.Exchange.MEXC.WSUrl, nil)
	if err != nil {
		return fmt.Errorf("mexc failed to dial websocket: %w", err)
	}

	err = c.subscribe(wsConn, symbols)
	if err != nil {
		return fmt.Errorf("mexc failed to subscribe to symbols: %w", err)
	}

	go c.readMessage(ctx, wsConn, category, outChan)
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
		payload := map[string]interface{}{
			"method": "sub.deal",
			"gzip":   false,
			"param": map[string]string{
				"symbol": denormalizeTickerName(symbol),
			},
		}

		if err := c.writeJSON(wsCon, payload); err != nil {
			return fmt.Errorf("mexc failed to send subscription request: %w", err)
		}
	}

	return nil
}

func (c *wsClient) readMessage(ctx context.Context, wsConn *websocket.Conn, category exchange.Category, outChan chan<- exchange.Trade) {
	defer func() {
		err := wsConn.Close()
		if err != nil && ctx.Err() == nil {
			c.logger.Warn().Err(err).Msg("mexc failed to close websocket connection")
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
			c.logger.Warn().Err(err).Msg("mexc failed to read message from MEXC websocket")
			// TODO: something wrong with connection need to reconnect (reconnect not implemented yet)
			return
		}

		if mt == websocket.TextMessage {
			var message dtos.WSBaseMessage

			if err = json.Unmarshal(messageByte, &message); err != nil {
				c.logger.Warn().Err(err).Msg("mexc failed to unmarshal base message from MEXC websocket")
			}

			if message.Channel != "push.deal" {
				continue
			}

			var tradeMessage dtos.WSTradeDTO
			if err = json.Unmarshal(messageByte, &tradeMessage); err != nil {
				c.logger.Warn().Err(err).Msg("mexc failed to unmarshal trade message from MEXC websocket")
			}

			symbol := normalizeTickerName(tradeMessage.Symbol)

			for _, val := range tradeMessage.Data {
				side := exchange.Buy
				if val.TradeSide == 2 {
					side = exchange.Sell
				}

				trade := exchange.Trade{
					Symbol:   symbol,
					Category: category,
					Ts:       val.TradeTime,
					Price:    val.Price,
					Volume:   val.Quantity,
					Side:     side,
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

func (c *wsClient) pingPongInterval(ctx context.Context, wsConn *websocket.Conn) {
	ticker := time.NewTicker(time.Second * 10)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			payload := map[string]string{
				"method": "ping",
			}
			err := c.writeJSON(wsConn, payload)
			if err != nil {
				c.logger.Warn().Err(err).Msg("mexc failed to send ping to MEXC websocket")
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
				c.logger.Warn().Msgf("MEXC metrics: dropped trades=%d", droppedTradesCnt)
			}
		}
	}
}
