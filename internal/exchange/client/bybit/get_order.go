package bybit

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
)

const getOrderURL = "/v5/order/realtime"

type getOrderResponseDTO struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  struct {
		List []struct {
			OrderID    string `json:"orderId"`
			AvgPrice   string `json:"avgPrice"`
			CumExecFee string `json:"cumExecFee"`
		} `json:"list"`
	} `json:"result"`
}

// GetOrder retrieves order details from ByBit by orderLinkId (our internal order UUID).
func (c *Client) GetOrder(ctx context.Context, orderID uuid.UUID, _ string, symbol string) (exchange.ExchangeOrder, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+getOrderURL, nil)
	if err != nil {
		return exchange.ExchangeOrder{}, fmt.Errorf("ByBit | GetOrder: failed to create request: %w", err)
	}

	query := req.URL.Query()
	query.Set("category", "linear")
	query.Set("symbol", symbol)
	query.Set("orderLinkId", orderID.String())
	req.URL.RawQuery = query.Encode()

	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	recvWindow := "5000"
	payload := timestamp + c.cfg.Exchange.ByBit.APIKey + recvWindow + req.URL.RawQuery
	signature := sign(c.cfg.Exchange.ByBit.APISecret, payload)
	c.setHeader(req, signature, timestamp, recvWindow)

	resp, err := c.http.Do(req)
	if err != nil {
		return exchange.ExchangeOrder{}, fmt.Errorf("ByBit | GetOrder: http request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return exchange.ExchangeOrder{}, fmt.Errorf("ByBit | GetOrder: failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return exchange.ExchangeOrder{}, fmt.Errorf("ByBit | GetOrder: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var raw getOrderResponseDTO
	if err := json.Unmarshal(body, &raw); err != nil {
		return exchange.ExchangeOrder{}, fmt.Errorf("ByBit | GetOrder: failed to unmarshal response: %w", err)
	}

	if raw.RetCode != 0 {
		return exchange.ExchangeOrder{}, fmt.Errorf("ByBit | GetOrder: API error, code: %d, msg: %s", raw.RetCode, raw.RetMsg)
	}

	if len(raw.Result.List) == 0 {
		return exchange.ExchangeOrder{}, fmt.Errorf("ByBit | GetOrder: order not found, orderLinkId: %s", orderID.String())
	}

	order := raw.Result.List[0]

	avgPrice, _ := decimal.NewFromString(order.AvgPrice)
	fees, _ := decimal.NewFromString(order.CumExecFee)

	return exchange.ExchangeOrder{
		OrderID:         orderID,
		ExchangeOrderID: order.OrderID,
		ExchangeName:    c.GetExchangeName(),
		AvgPrice:        avgPrice,
		Fees:            fees,
	}, nil
}
