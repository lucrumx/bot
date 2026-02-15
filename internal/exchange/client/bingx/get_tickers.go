package bingx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/lucrumx/bot/internal/exchange"
)

// GetTickers returns the latest ticker data for the specified symbol.
func (c *Client) GetTickers(ctx context.Context, symbols []string, category exchange.Category) ([]exchange.Ticker, error) {
	if category != exchange.CategoryLinear {
		return nil, fmt.Errorf("unsupported category")
	}

	// BingX API supports only one symbol at a time, but the interface requires a slice
	var symbol string
	if len(symbols) > 0 {
		symbol = symbols[0]
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
	if len(symbol) > 0 {
		if !strings.Contains(symbol, "-") {
			// BingX API requires the symbol to be in the format "BTC-USDT"
			symbol = strings.TrimSuffix(symbol, "USDT") + "-USDT"
		}
		fmt.Println("-----------------------: " + symbol)
	}

	queryStr := getSortedQuery(query, time.Now().UnixMilli(), false)
	signature := c.computeHmac256(queryStr)
	req.URL.RawQuery = fmt.Sprintf("%s&signature=%s", getSortedQuery(query, 0, true), signature)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("BingX client http request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("BingX client unexpected http while getting tickers status code: %d", resp.StatusCode)
	}

	var raw ResponseGetTickerDTO

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("BingX client failed to read response body: %w", err)
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("BingX client failed to unmarshal response: %w", err)
	}

	if raw.Code != 0 {
		return nil, fmt.Errorf("BingX client failed to get tickers, code: %d, msg: %s", raw.Code, raw.Msg)
	}

	result := make([]exchange.Ticker, 0, len(raw.Data))
	for _, dto := range raw.Data {
		t, err := mapTicker(dto)
		if err != nil {
			return nil, fmt.Errorf("BingX client failed to map response: %w", err)
		}

		result = append(result, t)
	}

	return result, nil
}
