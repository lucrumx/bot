package bingx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/lucrumx/bot/internal/exchange"
	"github.com/lucrumx/bot/internal/exchange/client/bingx/dtos"
)

// GetTickers returns the latest ticker data for the specified symbol.
func (c *Client) GetTickers(ctx context.Context, symbols []string, category exchange.Category) ([]exchange.Ticker, error) {
	if category != exchange.CategoryLinear {
		return nil, fmt.Errorf("unsupported category")
	}

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL+"/openApi/swap/v2/quote/contracts",
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("BingX client failed to create request: %w", err)
	}

	query := make(map[string]string)

	// BingX API supports only one symbol at a time, but the interface requires a slice
	if len(symbols) > 0 {
		query["symbol"] = denormalizeTickerName(symbols[0])
	}

	queryStr := getSortedQuery(query, time.Now().UnixMilli(), false)
	signature := computeHmac256(c.cfg, queryStr)
	req.URL.RawQuery = fmt.Sprintf("%s&signature=%s", getSortedQuery(query, 0, true), signature)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("BingX client http get tickers request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("BingX client unexpected http while getting tickers status code: %d", resp.StatusCode)
	}

	var raw dtos.ResponseGetTickerDTO

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("BingX client failed to read get tickers response body: %w", err)
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("BingX client failed to unmarshal get tickers response: %w", err)
	}

	if raw.Code != 0 {
		return nil, fmt.Errorf("BingX client failed to get tickers, code: %d, msg: %s", raw.Code, raw.Msg)
	}

	tickers, err := raw.ParseData()
	if err != nil {
		return nil, fmt.Errorf("BingX client failed to parse tickers data: %w", err)
	}

	result := make([]exchange.Ticker, 0, len(tickers))
	for _, dto := range tickers {
		t, err := mapTicker(dto)
		if err != nil {
			return nil, fmt.Errorf("BingX client failed to map get tickers response: %w", err)
		}

		result = append(result, t)
	}

	return result, nil
}
