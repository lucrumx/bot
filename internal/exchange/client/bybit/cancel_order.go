package bybit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
)

// API: POST /v5/order/cancel
// Docs: https://bybit-exchange.github.io/docs/v5/order/cancel-order

const cancelOrderURL = "/v5/order/cancel"

// CancelOrder cancels a pending limit order by orderLinkId (our internal UUID).
func (c *Client) CancelOrder(ctx context.Context, orderID uuid.UUID, _ string, symbol string) error {
	payload := map[string]interface{}{
		"category":    "linear",
		"symbol":      symbol,
		"orderLinkId": orderID.String(),
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("ByBit | CancelOrder: failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+cancelOrderURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("ByBit | CancelOrder: failed to create request: %w", err)
	}

	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	recvWindow := strconv.FormatInt(c.cfg.Exchange.ByBit.RecvWindow, 10)
	payloadStr := string(bodyBytes)
	signature := sign(c.cfg.Exchange.ByBit.APISecret, timestamp+c.cfg.Exchange.ByBit.APIKey+recvWindow+payloadStr)

	req.Header.Set("Content-Type", "application/json")
	c.setHeader(req, signature, timestamp, recvWindow)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("ByBit | CancelOrder: http request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ByBit | CancelOrder: failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ByBit | CancelOrder: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var raw struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return fmt.Errorf("ByBit | CancelOrder: failed to unmarshal response: %w", err)
	}

	if raw.RetCode != 0 {
		return fmt.Errorf("ByBit | CancelOrder: API error, code: %d, msg: %s", raw.RetCode, raw.RetMsg)
	}

	return nil
}
