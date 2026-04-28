package bingx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/shopspring/decimal"

	"github.com/lucrumx/bot/internal/exchange"
)

// GetInstruments retrieves contract specifications for all instruments from BingX.
func (c *Client) GetInstruments(ctx context.Context) (map[string]exchange.Instrument, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/openApi/swap/v2/quote/contracts", nil)
	if err != nil {
		return nil, fmt.Errorf("BingX GetInstruments: failed to create request: %w", err)
	}

	queryStr := getSortedQuery(nil, time.Now().UnixMilli(), false)
	signature := computeHmac256(c.cfg, queryStr)
	req.URL.RawQuery = fmt.Sprintf("%s&signature=%s", getSortedQuery(nil, 0, true), signature)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("BingX GetInstruments: http request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("BingX GetInstruments: failed to read body: %w", err)
	}

	var raw struct {
		Code int64  `json:"code"`
		Msg  string `json:"msg"`
		Data []struct {
			Symbol            string `json:"symbol"`
			DisplayName       string `json:"displayName"`
			Size              string `json:"size"`
			PricePrecision    int64  `json:"pricePrecision"`
			QuantityPrecision int64  `json:"quantityPrecision"`
			Status            int64  `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("BingX GetInstruments: failed to unmarshal: %w", err)
	}

	if raw.Code != 0 {
		return nil, fmt.Errorf("BingX GetInstruments: API error, code: %d, msg: %s", raw.Code, raw.Msg)
	}

	result := make(map[string]exchange.Instrument, len(raw.Data))
	for _, dto := range raw.Data {
		if dto.Status != 1 {
			continue
		}
		volStep, err := decimal.NewFromString(dto.Size)
		if err != nil {
			return nil, fmt.Errorf("BingX GetInstruments: invalid size for %s: %w", dto.Symbol, err)
		}

		symbol := normalizeTickerName(dto.DisplayName)

		result[symbol] = exchange.Instrument{
			Symbol:       symbol,
			VolStep:      volStep,
			MinVol:       volStep,
			PriceStep:    decimal.New(1, -int32(dto.PricePrecision)),
			ContractSize: decimal.NewFromInt(1),
		}
	}

	return result, nil
}
