package mexc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/shopspring/decimal"

	"github.com/lucrumx/bot/internal/exchange"
)

// GetInstruments retrieves contract specifications for all instruments from MEXC.
func (c *Client) GetInstruments(ctx context.Context) (map[string]exchange.Instrument, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v1/contract/detail", nil)
	if err != nil {
		return nil, fmt.Errorf("MEXC GetInstruments: failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("MEXC GetInstruments: http request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("MEXC GetInstruments: failed to read body: %w", err)
	}

	var raw struct {
		Success bool `json:"success"`
		Code    int  `json:"code"`
		Data    []struct {
			Symbol       string  `json:"symbol"`
			State        int     `json:"state"` // 0 = enabled
			VolUnit      float64 `json:"volUnit"`
			MinVol       float64 `json:"minVol"`
			PriceUnit    float64 `json:"priceUnit"`
			ContractSize float64 `json:"contractSize"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("MEXC GetInstruments: failed to unmarshal: %w", err)
	}

	if !raw.Success || raw.Code != 0 {
		return nil, fmt.Errorf("MEXC GetInstruments: API error, success: %t, code: %d", raw.Success, raw.Code)
	}

	result := make(map[string]exchange.Instrument, len(raw.Data))
	for _, item := range raw.Data {
		if item.State != 0 {
			continue
		}
		symbol := normalizeTickerName(item.Symbol)

		contractSize := decimal.NewFromFloat(item.ContractSize)
		if contractSize.IsZero() {
			contractSize = decimal.NewFromInt(1)
		}

		result[symbol] = exchange.Instrument{
			Symbol:       symbol,
			VolStep:      decimal.NewFromFloat(item.VolUnit),
			MinVol:       decimal.NewFromFloat(item.MinVol),
			PriceStep:    decimal.NewFromFloat(item.PriceUnit),
			ContractSize: contractSize,
		}
	}

	return result, nil
}
