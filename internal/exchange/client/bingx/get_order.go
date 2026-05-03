package bingx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/lucrumx/bot/internal/exchange"
	"github.com/lucrumx/bot/internal/exchange/client/bingx/dtos"
)

const (
	getOrderURL = "/openApi/swap/v2/trade/order"
)

// GetOrder retrieves the details of an order from the exchange using its exchange order ID and symbol. It returns an ExchangeOrder struct containing the average price, fees, and other relevant information about the order.
func (c *Client) GetOrder(ctx context.Context, orderID uuid.UUID, exchangeOrderID string, symbol string) (exchange.ExchangeOrder, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+getOrderURL, nil)
	if err != nil {
		return exchange.ExchangeOrder{}, fmt.Errorf("BingX client failed to create get order request: %w", err)
	}

	query := make(map[string]string)
	query["orderId"] = exchangeOrderID
	query["symbol"] = denormalizeTickerName(symbol)
	timestamp := time.Now().UnixMilli()
	queryStr := getSortedQuery(query, timestamp, false)
	signature := computeHmac256(c.cfg, queryStr)
	req.URL.RawQuery = fmt.Sprintf("%s&signature=%s", getSortedQuery(query, timestamp, true), signature)

	req.Header.Set("X-BX-APIKEY", c.cfg.Exchange.BingX.APIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return exchange.ExchangeOrder{}, fmt.Errorf("BingX client http balance request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return exchange.ExchangeOrder{}, fmt.Errorf("BingX client unexpected http while getting balance, status code: %d", resp.StatusCode)
	}

	var raw dtos.GetOrderResponseDTO

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return exchange.ExchangeOrder{}, fmt.Errorf("BingX client failed to read get balance response body: %w", err)
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		return exchange.ExchangeOrder{}, fmt.Errorf("BingX client failed to unmarshal get balance response: %w", err)
	}

	if raw.Code != 0 {
		return exchange.ExchangeOrder{}, fmt.Errorf("BingX client failed to get balance, code: %d, msg: %s", raw.Code, raw.Msg)
	}

	return exchange.ExchangeOrder{
		OrderID:         orderID,
		ExchangeOrderID: exchangeOrderID,
		ExchangeName:    c.GetExchangeName(),
		AvgPrice:        raw.Data.Order.AvgPrice.Decimal,
		Fees:            raw.Data.Order.Commission.Decimal,
	}, nil
}
