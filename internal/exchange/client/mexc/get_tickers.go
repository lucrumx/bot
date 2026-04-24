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
		s := make([]string, 0, len(symbols))
		for _, symbol := range symbols {
			s = append(s, denormalizeTickerName(symbol))
		}
		q.Set("symbol", strings.Join(s, ","))
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

	tickers, err := parseTickers(data.Data)
	if err != nil {
		return nil, fmt.Errorf("MEXC client failed to parse tickers: %w", err)
	}

	result := make([]exchange.Ticker, 0, len(tickers))
	for _, t := range tickers {
		if !strings.Contains(t.Symbol, "_USDT") {
			continue
		}
		result = append(result, mapTicker(&t))
	}

	return result, nil
}

// parseTickers handles both array and single-object responses from MEXC ticker API.
func parseTickers(raw json.RawMessage) ([]dtos.TickerDTO, error) {
	var list []dtos.TickerDTO
	if err := json.Unmarshal(raw, &list); err == nil {
		return list, nil
	}

	var single dtos.TickerDTO
	if err := json.Unmarshal(raw, &single); err != nil {
		return nil, err
	}
	return []dtos.TickerDTO{single}, nil
}

func mapTicker(ticker *dtos.TickerDTO) exchange.Ticker {
	return exchange.Ticker{
		Symbol:    normalizeTickerName(ticker.Symbol),
		LastPrice: decimal.NewFromFloat(ticker.LastPrice),
	}
}
