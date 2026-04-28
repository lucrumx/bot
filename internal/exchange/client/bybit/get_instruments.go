package bybit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/shopspring/decimal"

	"github.com/lucrumx/bot/internal/exchange"
)

// GetInstruments retrieves contract specifications for all linear instruments from ByBit.
func (c *Client) GetInstruments(ctx context.Context) (map[string]exchange.Instrument, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v5/market/instruments-info", nil)
	if err != nil {
		return nil, fmt.Errorf("ByBit GetInstruments: failed to create request: %w", err)
	}

	q := req.URL.Query()
	q.Set("category", "linear")
	q.Set("limit", "1000")
	req.URL.RawQuery = q.Encode()

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ByBit GetInstruments: http request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ByBit GetInstruments: failed to read body: %w", err)
	}

	var raw response[struct {
		List []struct {
			Symbol        string `json:"symbol"`
			LotSizeFilter struct {
				QtyStep     string `json:"qtyStep"`
				MinOrderQty string `json:"minOrderQty"`
			} `json:"lotSizeFilter"`
			PriceFilter struct {
				TickSize string `json:"tickSize"`
			} `json:"priceFilter"`
		} `json:"list"`
	}]

	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("ByBit GetInstruments: failed to unmarshal: %w", err)
	}

	if raw.RetCode != 0 {
		return nil, &apiError{Code: raw.RetCode, Message: raw.RetMsg}
	}

	result := make(map[string]exchange.Instrument, len(raw.Result.List))
	for _, item := range raw.Result.List {
		volStep, err := decimal.NewFromString(item.LotSizeFilter.QtyStep)
		if err != nil {
			return nil, fmt.Errorf("ByBit GetInstruments: invalid qtyStep for %s: %w", item.Symbol, err)
		}
		minVol, err := decimal.NewFromString(item.LotSizeFilter.MinOrderQty)
		if err != nil {
			return nil, fmt.Errorf("ByBit GetInstruments: invalid minOrderQty for %s: %w", item.Symbol, err)
		}
		priceStep, err := decimal.NewFromString(item.PriceFilter.TickSize)
		if err != nil {
			return nil, fmt.Errorf("ByBit GetInstruments: invalid tickSize for %s: %w", item.Symbol, err)
		}
		result[item.Symbol] = exchange.Instrument{
			Symbol:       item.Symbol,
			VolStep:      volStep,
			MinVol:       minVol,
			PriceStep:    priceStep,
			ContractSize: decimal.NewFromInt(1),
		}
	}

	return result, nil
}
