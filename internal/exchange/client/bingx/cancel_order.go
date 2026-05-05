package bingx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// API: DELETE /openApi/swap/v2/trade/order
// Docs: https://bingx-api.github.io/docs-v3/#/en/Swap/Trades%20Endpoints/Cancel%20Order

const cancelOrderURL = "/openApi/swap/v2/trade/order"

// CancelOrder cancels a pending limit order by clientOrderId.
func (c *Client) CancelOrder(ctx context.Context, orderID uuid.UUID, _ string, symbol string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+cancelOrderURL, nil)
	if err != nil {
		return fmt.Errorf("BingX | CancelOrder: failed to create request: %w", err)
	}

	timestamp := time.Now().UnixMilli()
	query := map[string]string{
		"symbol":        denormalizeTickerName(symbol),
		"clientOrderId": orderID.String(),
	}

	queryStr := getSortedQuery(query, timestamp, false)
	signature := computeHmac256(c.cfg, queryStr)
	req.URL.RawQuery = fmt.Sprintf("%s&signature=%s", getSortedQuery(query, timestamp, true), signature)
	req.Header.Set("X-BX-APIKEY", c.cfg.Exchange.BingX.APIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("BingX | CancelOrder: http request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("BingX | CancelOrder: failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("BingX | CancelOrder: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var raw struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return fmt.Errorf("BingX | CancelOrder: failed to unmarshal response: %w", err)
	}

	if raw.Code != 0 {
		return fmt.Errorf("BingX | CancelOrder: API error, code: %d, msg: %s", raw.Code, raw.Msg)
	}

	return nil
}
