package mexc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/lucrumx/bot/internal/exchange"
	"github.com/lucrumx/bot/internal/exchange/client/mexc/dtos"
)

// GetTickers retrieves tickers from MEXC API.
func (c *Client) GetTickers(ctx context.Context, symbols []string, category exchange.Category) ([]exchange.Ticker, error) {
	if category != exchange.CategoryLinear {
		return nil, fmt.Errorf("unsupported category")
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL+"/api/v1/contract/ticker",
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("MOEXC client failed to create request: %w", err)
	}

	q := req.URL.Query()
	if len(symbols) > 0 {
		q.Set("symbols", strings.Join(symbols, ","))
	}

	req.URL.RawQuery = q.Encode()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("MOEXC client http request failed: %w", err)
	}

	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("MOEXC client unexpected http while getting tickers status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("MOEXC client failed to read response body: %w", err)
	}

	var data dtos.GetTickersDTO
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("MOEXC client failed to unmarshal response: %w", err)
	}

	if !data.Success || data.Code != 0 {
		return nil, fmt.Errorf("MOEXC client failed to get tickers, success: %t, code: %d", data.Success, data.Code)
	}

	result := make([]exchange.Ticker, 0, len(data.Data))
	for _, data := range data.Data {
		if !strings.Contains(data.Symbol, "_USDT") {
			continue
		}

		result = append(result, mapTicker(&data))
	}

	return result, nil
}

func mapTicker(ticker *dtos.TickerDTO) exchange.Ticker {
	return exchange.Ticker{
		Symbol:    normalizeTickerName(ticker.Symbol),
		LastPrice: decimal.NewFromFloat(ticker.LastPrice),
	}
}
