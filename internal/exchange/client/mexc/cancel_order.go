package mexc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/google/uuid"
)

// API: POST /api/v1/private/order/cancel
// Docs: https://www.mexc.com/api-docs/futures/account-and-trading-endpoints/cancel-orders
//
// Body is a bare JSON array of MEXC-assigned order IDs (Long), up to 50 per call:
//   [101716841474621953, 108885377779302912]
//
// We always cancel one order at a time. The UUID parameter is ignored — MEXC requires its own
// orderID, which is captured into Leg.ExchangeOrderID after CreateOrder returns. The watcher in
// engine_fill_timeout.go pulls it from OpenTimeoutInfo and passes it here.

const cancelOrderURL = "/api/v1/private/order/cancel"

// CancelOrder cancels a pending limit order by the exchange-assigned order ID.
func (c *Client) CancelOrder(ctx context.Context, _ uuid.UUID, exchangeOrderID string, _ string) error {
	if exchangeOrderID == "" {
		return fmt.Errorf("MEXC | CancelOrder: exchangeOrderID is empty (CreateOrder may not have completed)")
	}

	orderIDInt, err := strconv.ParseInt(exchangeOrderID, 10, 64)
	if err != nil {
		return fmt.Errorf("MEXC | CancelOrder: invalid exchangeOrderID %q: %w", exchangeOrderID, err)
	}

	payload := []int64{orderIDInt}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("MEXC | CancelOrder: failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+cancelOrderURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("MEXC | CancelOrder: failed to create request: %w", err)
	}

	setSignedHeaders(req, c.cfg.Exchange.MEXC.APIKey, c.cfg.Exchange.MEXC.APISecret, string(bodyBytes))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("MEXC | CancelOrder: http request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("MEXC | CancelOrder: failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("MEXC | CancelOrder: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var raw struct {
		Success bool `json:"success"`
		Code    int  `json:"code"`
		Data    []struct {
			OrderID   int64  `json:"orderId"`
			ErrorCode int    `json:"errorCode"`
			ErrorMsg  string `json:"errorMsg"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return fmt.Errorf("MEXC | CancelOrder: failed to unmarshal response: %w", err)
	}

	if !raw.Success || raw.Code != 0 {
		return fmt.Errorf("MEXC | CancelOrder: API error, success: %t, code: %d, body: %s", raw.Success, raw.Code, string(body))
	}

	// Per-order error code in data[0].errorCode — non-zero means the cancel was rejected for this order.
	if len(raw.Data) > 0 && raw.Data[0].ErrorCode != 0 {
		return fmt.Errorf("MEXC | CancelOrder: order rejected, errorCode: %d, errorMsg: %s", raw.Data[0].ErrorCode, raw.Data[0].ErrorMsg)
	}

	return nil
}
