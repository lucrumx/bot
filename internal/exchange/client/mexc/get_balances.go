package mexc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/shopspring/decimal"

	"github.com/lucrumx/bot/internal/models"
)

// GetBalances returns the balances of the user's account on the exchange.
func (c *Client) GetBalances(ctx context.Context) ([]models.Balance, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/private/account/assets", nil)
	if err != nil {
		return nil, fmt.Errorf("MEXC | GetBalances: failed to create request: %w", err)
	}

	setSignedHeaders(req, c.cfg.Exchange.MEXC.APIKey, c.cfg.Exchange.MEXC.APISecret, "")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("MEXC | GetBalances: http request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("MEXC | GetBalances: failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MEXC | GetBalances: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var raw struct {
		Success bool `json:"success"`
		Code    int  `json:"code"`
		Data    []struct {
			Currency         string  `json:"currency"`
			AvailableBalance float64 `json:"availableBalance"`
			PositionMargin   float64 `json:"positionMargin"`
			Equity           float64 `json:"equity"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("MEXC | GetBalances: failed to unmarshal response: %w", err)
	}

	if !raw.Success || raw.Code != 0 {
		return nil, fmt.Errorf("MEXC | GetBalances: API error, success: %t, code: %d", raw.Success, raw.Code)
	}

	result := make([]models.Balance, 0, len(raw.Data))
	for _, b := range raw.Data {
		result = append(result, models.Balance{
			ExchangeName: c.GetExchangeName(),
			Asset:        b.Currency,
			Free:         decimal.NewFromFloat(b.AvailableBalance),
			Locked:       decimal.NewFromFloat(b.PositionMargin),
			Total:        decimal.NewFromFloat(b.Equity),
		})
	}

	return result, nil
}
