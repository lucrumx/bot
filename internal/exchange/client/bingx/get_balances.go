package bingx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/lucrumx/bot/internal/exchange/client/bingx/dtos"
)

const balanceURL = "/openApi/swap/v3/user/balance"

// GetBalances returns the balances of the user's account on the exchange.
func (c *Client) GetBalances(ctx context.Context) ([]dtos.Balance, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL+balanceURL,
		nil)

	if err != nil {
		return nil, fmt.Errorf("BingX client failed to create request: %w", err)
	}

	query := make(map[string]string)
	timestamp := time.Now().UnixMilli()
	queryStr := getSortedQuery(query, timestamp, false)
	signature := c.computeHmac256(queryStr)
	req.URL.RawQuery = fmt.Sprintf("%s&signature=%s", getSortedQuery(query, timestamp, true), signature)

	req.Header.Set("X-BX-APIKEY", c.cfg.Exchange.BingX.APIKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("BingX client http balance request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("BingX client unexpected http while getting balance, status code: %d", resp.StatusCode)
	}

	var raw dtos.ResponseGetBalanceDTO

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("BingX client failed to read get balance response body: %w", err)
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("BingX client failed to unmarshal get balance response: %w", err)
	}

	if raw.Code != 0 {
		return nil, fmt.Errorf("BingX client failed to get balance, code: %d, msg: %s", raw.Code, raw.Msg)
	}

	return raw.Data, nil
}
