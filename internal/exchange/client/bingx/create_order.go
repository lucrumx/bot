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

	"github.com/lucrumx/bot/internal/exchange/client/bingx/dtos"
	"github.com/lucrumx/bot/internal/models"
)

// Api description: https://bingx-api.github.io/docs-v3/#/en/Swap/Trades%20Endpoints/Place%20order

const orderURL = "/openApi/swap/v2/trade/order/test"

// CreateOrder creates an order on the exchange.
func (c *Client) CreateOrder(ctx context.Context, order models.Order) error {
	if err := validateBeforeCreateOrder(&order); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+orderURL,
		nil)

	if err != nil {
		return fmt.Errorf("BingX client failed to create request: %w", err)
	}

	timestamp := time.Now().UnixMilli()

	query := mapRequestDataToOrderDTO(&order, timestamp)
	queryStr := getSortedQuery(query, timestamp, false)
	signature := c.computeHmac256(queryStr)
	req.URL.RawQuery = fmt.Sprintf("%s&signature=%s", getSortedQuery(query, timestamp, true), signature)

	req.Header.Set("X-BX-APIKEY", c.cfg.Exchange.BingX.APIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("BingX client http create order request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("BingX client unexpected http while creating order, status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("BingX client failed to read create order response body: %w", err)
	}

	var raw dtos.OrderCreateResponseDTO

	if err := json.Unmarshal(body, &raw); err != nil {
		return fmt.Errorf("BingX client failed to unmarshal create order response: %w", err)
	}

	if raw.Code != 0 {
		return fmt.Errorf("BingX client failed to create order, code: %d, msg: %s", raw.Code, raw.Msg)
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
