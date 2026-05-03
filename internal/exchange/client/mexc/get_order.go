package mexc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/google/uuid"

	"github.com/lucrumx/bot/internal/exchange"
	"github.com/lucrumx/bot/internal/exchange/client/mexc/dtos"
)

const getOrderURL = "/api/v1/private/order/get/"

// GetOrder returns the balances of the user's account on the exchange.
func (c *Client) GetOrder(ctx context.Context, orderID uuid.UUID, exchangeOrderID string, _ string) (exchange.ExchangeOrder, error) {
	if exchangeOrderID == "" {
		return exchange.ExchangeOrder{}, fmt.Errorf("MEXC | GetOrder: exchangeOrderID is empty")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+getOrderURL+"/"+exchangeOrderID, nil)
	if err != nil {
		return exchange.ExchangeOrder{}, fmt.Errorf("MEXC | GetOrder: failed to create request: %w", err)
	}

	setSignedHeaders(req, c.cfg.Exchange.MEXC.APIKey, c.cfg.Exchange.MEXC.APISecret, "")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return exchange.ExchangeOrder{}, fmt.Errorf("MEXC | GetOrder: http request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return exchange.ExchangeOrder{}, fmt.Errorf("MEXC | GetOrder: failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return exchange.ExchangeOrder{}, fmt.Errorf("MEXC | GetOrder: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var raw dtos.GetOrderResponseDTO

	if err := json.Unmarshal(body, &raw); err != nil {
		return exchange.ExchangeOrder{}, fmt.Errorf("MEXC | GetOrder: failed to unmarshal response: %w", err)
	}

	fmt.Printf("%+v\n", raw)

	if !raw.Success || raw.Code != 0 {
		return exchange.ExchangeOrder{}, fmt.Errorf("MEXC | GetOrder: API error, success: %t, code: %d", raw.Success, raw.Code)
	}

	return exchange.ExchangeOrder{
		OrderID:         orderID,
		ExchangeOrderID: exchangeOrderID,
		ExchangeName:    c.GetExchangeName(),
		AvgPrice:        raw.Data.DealAvgPrice.Decimal,
		Fees:            raw.Data.TotalFee.Decimal,
	}, nil
}
