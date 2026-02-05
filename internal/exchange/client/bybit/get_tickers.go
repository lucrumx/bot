package bybit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/lucrumx/bot/internal/exchange"
)

type apiError struct {
	Code    int
	Message string
}

func (e *apiError) Error() string {
	return fmt.Sprintf("ByBit api error, %d, %s", e.Code, e.Message)
}

type response[T any] struct {
	RetCode int    `json:"retCode"`
	RetMsg  string `json:"retMsg"`
	Result  T      `json:"result"`
}

// GetTickers retrieves tickers from ByBit API.
func (c *Client) GetTickers(ctx context.Context, symbols []string, category exchange.Category) ([]exchange.Ticker, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL+"/v5/market/tickers",
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("ByBit client failed to create request: %w", err)
	}

	q := req.URL.Query()
	if len(symbols) > 0 {
		q.Set("symbols", strings.Join(symbols, ","))
	}
	q.Set("category", string(category))
	req.URL.RawQuery = q.Encode()

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ByBit client http request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	var raw response[struct {
		List []TickerDTO `json:"list"`
	}]

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ByBit client failed to read response body: %w", err)
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("ByBit client failed to unmarshal response: %w", err)
	}

	if raw.RetCode != 0 {
		return nil, &apiError{Code: raw.RetCode, Message: raw.RetMsg}
	}

	result := make([]exchange.Ticker, 0, len(raw.Result.List))
	for _, dto := range raw.Result.List {
		t, err := mapTicker(dto)
		if err != nil {
			return nil, fmt.Errorf("ByBit client failed to map response: %w", err)
		}

		result = append(result, t)
	}

	return result, nil
}
