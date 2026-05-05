package mexc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"
)

// API: DELETE /api/v1/private/order/cancel/{orderId}
// Docs: https://mexcdevelop.github.io/apidocs/contract_v1_en/#cancel-the-order

const cancelOrderURL = "/api/v1/private/order/cancel/"

// CancelOrder cancels a pending limit order by exchange order ID.
func (c *Client) CancelOrder(ctx context.Context, _ uuid.UUID, exchangeOrderID string, _ string) error {
	if exchangeOrderID == "" {
		return fmt.Errorf("MEXC | CancelOrder: exchangeOrderID is required")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, c.baseURL+cancelOrderURL+exchangeOrderID, nil)
	if err != nil {
		return fmt.Errorf("MEXC | CancelOrder: failed to create request: %w", err)
	}

	setSignedHeaders(req, c.cfg.Exchange.MEXC.APIKey, c.cfg.Exchange.MEXC.APISecret, "")

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
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return fmt.Errorf("MEXC | CancelOrder: failed to unmarshal response: %w", err)
	}

	if !raw.Success || raw.Code != 0 {
		return fmt.Errorf("MEXC | CancelOrder: API error, success: %t, code: %d", raw.Success, raw.Code)
	}

	return nil
}
