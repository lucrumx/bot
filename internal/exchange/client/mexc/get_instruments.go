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
			Symbol            string  `json:"symbol"`
			State             int     `json:"state"`             // 0 enabled, 1 delivery, 2 delivered, 3 offline, 4 paused
			APIAllowed        bool    `json:"apiAllowed"`        // flips to false only on the final delisting day
			Type              int     `json:"type"`              // 1 normal, 2 suspended
			FutureType        int     `json:"futureType"`        // 1 perpetual, 2 delivery
			AutomaticDelivery int     `json:"automaticDelivery"` // for futureType=1: 0 normal, 1 scheduled for delivery (= pre-delisting)
			DeliveryTime      int64   `json:"deliveryTime"`      // ms, set when AutomaticDelivery=1
			VolUnit           float64 `json:"volUnit"`
			MinVol            float64 `json:"minVol"`
			PriceUnit         float64 `json:"priceUnit"`
			ContractSize      float64 `json:"contractSize"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("MEXC GetInstruments: failed to unmarshal: %w", err)
	}

	if !raw.Success || raw.Code != 0 {
		return nil, fmt.Errorf("MEXC GetInstruments: API error, success: %t, code: %d", raw.Success, raw.Code)
	}

	result := make(map[string]exchange.Instrument, len(raw.Data))
	var skippedState, skippedAPI, skippedType, skippedPreDelisting int
	for _, item := range raw.Data {
		// MEXC keeps `state == 0` (Enabled) during pre-delisting wind-down so existing
		// positions can be closed. New orders get rejected with code 8823.
		if item.State != 0 {
			skippedState++
			continue
		}
		if !item.APIAllowed {
			skippedAPI++
			continue
		}
		if item.Type != 1 {
			skippedType++
			continue
		}
		// Pre-delisting marker: a perpetual (futureType=1) that has been scheduled for delivery
		// (automaticDelivery=1) is in wind-down. CreateOrder will return code 8823 even though
		// apiAllowed is still true and state is still 0. This catches the announcement window
		// that the other flags miss.
		if item.FutureType == 1 && item.AutomaticDelivery == 1 {
			skippedPreDelisting++
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

	c.logger.Info().
		Int("kept", len(result)).
		Int("skipped_state", skippedState).
		Int("skipped_api_disabled", skippedAPI).
		Int("skipped_suspended", skippedType).
		Int("skipped_pre_delisting", skippedPreDelisting).
		Msg("MEXC instruments loaded")

	return result, nil
}
