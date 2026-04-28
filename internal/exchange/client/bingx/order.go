package bingx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/lucrumx/bot/internal/exchange"
	"github.com/lucrumx/bot/internal/exchange/client/bingx/dtos"
	"github.com/lucrumx/bot/internal/models"
)

// Api description: https://bingx-api.github.io/docs-v3/#/en/Swap/Trades%20Endpoints/Place%20order

const orderURL = "/openApi/swap/v2/trade/order"

// CreateOrder sends a market order to the exchange.
// On success, mutates order: sets ExchangeOrderID, ExchangeName, Status, RawResponse.
func (c *Client) CreateOrder(ctx context.Context, order *models.Order) error {
	if err := validateBeforeCreateOrder(order); err != nil {
		return err
	}

	timestamp := time.Now().UnixMilli()
	return c.submitOrder(ctx, order, mapRequestDataToOrderDTO(order, timestamp), timestamp)
}

// CloseOrder closes an existing position by placing an order in the opposite direction with the same positionSide.
func (c *Client) CloseOrder(ctx context.Context, order *models.Order) error {
	if err := validateBeforeCreateOrder(order); err != nil {
		return err
	}

	// flip side, keep positionSide — this closes the position
	side := dtos.OrderSideSell
	positionSide := dtos.OrderPositionSideLong
	if order.Side == models.OrderSideSell {
		side = dtos.OrderSideBuy
		positionSide = dtos.OrderPositionSideShort
	}

	timestamp := time.Now().UnixMilli()
	query := map[string]string{
		"symbol":        denormalizeTickerName(order.Symbol),
		"side":          string(side),
		"positionSide":  string(positionSide),
		"type":          string(dtos.OrderTypeMarket),
		"quantity":      order.Quantity.String(),
		"clientOrderId": order.ID.String(),
		"timestamp":     strconv.FormatInt(timestamp, 10),
	}

	return c.submitOrder(ctx, order, query, timestamp)
}

func (c *Client) submitOrder(ctx context.Context, order *models.Order, query map[string]string, timestamp int64) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+orderURL, nil)
	if err != nil {
		return fmt.Errorf("BingX client failed to create request: %w", err)
	}

	queryStr := getSortedQuery(query, timestamp, false)
	signature := computeHmac256(c.cfg, queryStr)
	req.URL.RawQuery = fmt.Sprintf("%s&signature=%s", getSortedQuery(query, timestamp, true), signature)
	req.Header.Set("X-BX-APIKEY", c.cfg.Exchange.BingX.APIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("BingX client http order request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("BingX client unexpected http status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("BingX client failed to read order response body: %w", err)
	}

	var raw dtos.OrderCreateResponseDTO
	if err := json.Unmarshal(body, &raw); err != nil {
		return fmt.Errorf("BingX client failed to unmarshal order response: %w", err)
	}

	if raw.Code != 0 {
		return fmt.Errorf("BingX client order failed, code: %d, msg: %s", raw.Code, raw.Msg)
	}

	confirmed := order
	confirmed.ExchangeName = c.GetExchangeName()
	if raw.Data.Order.OrderID > 0 {
		confirmed.ExchangeOrderID = strconv.FormatInt(raw.Data.Order.OrderID, 10)
	}
	confirmed.RawResponse = string(body)
	confirmed.Status = models.OrderStatusPending

	// BingX не шлёт execution events через WS — подтверждаем из REST ответа
	if raw.Data.Order.Status == "FILLED" && c.wsPrivate != nil {
		execPrice, _ := decimal.NewFromString(raw.Data.Order.AvgPrice)
		execQty, _ := decimal.NewFromString(raw.Data.Order.ExecutedQty)

		c.wsPrivate.executionChannel <- exchange.OrderExecutionEvent{
			OrderID:   order.ID,
			ExecPrice: execPrice,
			ExecQty:   execQty,
			ExecValue: execPrice.Mul(execQty),
			OrderQty:  order.Quantity,
		}
	}

	return nil
}

func validateBeforeCreateOrder(order *models.Order) error {
	if order.Type != models.OrderTypeMarket {
		return fmt.Errorf("BingX client support only market orders")
	}

	if order.Quantity.LessThanOrEqual(decimal.NewFromInt(0)) {
		return fmt.Errorf("BingX client order quantity must be greater than 0")
	}

	if len(order.Symbol) <= 0 {
		return fmt.Errorf("BingX client order symbol must be specified")
	}

	if order.ID == uuid.Nil {
		return fmt.Errorf("BingX client order id must be valid uuid")
	}

	return nil
}

func mapRequestDataToOrderDTO(order *models.Order, timestamp int64) map[string]string {
	side := dtos.OrderSideBuy
	positionSide := dtos.OrderPositionSideLong

	if order.Side == models.OrderSideSell {
		side = dtos.OrderSideSell
		positionSide = dtos.OrderPositionSideShort
	}
	orderType := dtos.OrderTypeMarket
	if order.Type == models.OrderTypeLimit {
		orderType = dtos.OrderTypeLimit
	}

	return map[string]string{
		"symbol":        denormalizeTickerName(order.Symbol),
		"side":          string(side),
		"positionSide":  string(positionSide),
		"type":          string(orderType),
		"quantity":      order.Quantity.String(),
		"clientOrderId": order.ID.String(),
		"timestamp":     strconv.FormatInt(timestamp, 10),
	}
}
