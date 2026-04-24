package mexc

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/lucrumx/bot/internal/models"
)

const (
	// order sides
	mexcSideOpenLong   = 1 // Buy: open long
	mexcSideCloseShort = 2 // Buy: close short
	mexcSideOpenShort  = 3 // Sell: open short
	mexcSideCloseLong  = 4 // Sell: close long

	// order types
	mexcOrderTypeMarket = 5

	// margin modes
	mexcOpenTypeCross = 2

	createOrderURL = "/api/v1/private/order/create"
)

// CreateOrder creates an order on the MEXC exchange.
func (c *Client) CreateOrder(ctx context.Context, order *models.Order) error {
	if err := validateOrder(order); err != nil {
		return err
	}

	side := mexcSideOpenLong
	if order.Side == models.OrderSideSell {
		side = mexcSideOpenShort
	}

	return c.submitOrder(ctx, order, side)
}

// CloseOrder closes an existing position on the MEXC exchange.
func (c *Client) CloseOrder(ctx context.Context, order *models.Order) error {
	if err := validateOrder(order); err != nil {
		return err
	}

	// Buy means we had a long position → close long (side=4)
	// Sell means we had a short position → close short (side=2)
	side := mexcSideCloseLong
	if order.Side == models.OrderSideSell {
		side = mexcSideCloseShort
	}

	return c.submitOrder(ctx, order, side)
}

func (c *Client) submitOrder(ctx context.Context, order *models.Order, side int) error {
	payload := map[string]interface{}{
		"symbol":      denormalizeTickerName(order.Symbol),
		"price":       0,
		"vol":         order.Quantity.String(),
		"side":        side,
		"type":        mexcOrderTypeMarket,
		"openType":    mexcOpenTypeCross,
		"externalOid": strings.ReplaceAll(order.ID.String(), "-", ""),
	}

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("MEXC | submitOrder: failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.createOrderURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("MEXC | submitOrder: failed to create request: %w", err)
	}

	setSignedHeaders(req, c.cfg.Exchange.MEXC.APIKey, c.cfg.Exchange.MEXC.APISecret, string(bodyBytes))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("MEXC | submitOrder: http request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("MEXC | submitOrder: failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("MEXC | submitOrder: unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var raw struct {
		Success bool `json:"success"`
		Code    int  `json:"code"`
		Data    struct {
			OrderID string `json:"orderId"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &raw); err != nil {
		return fmt.Errorf("MEXC | submitOrder: failed to unmarshal response: %w", err)
	}

	if !raw.Success || raw.Code != 0 {
		return fmt.Errorf("MEXC | submitOrder: API error, success: %t, code: %d, body: %s", raw.Success, raw.Code, string(body))
	}

	confirmed := order
	confirmed.ExchangeName = c.GetExchangeName()
	confirmed.ExchangeOrderID = raw.Data.OrderID
	confirmed.RawResponse = string(body)
	confirmed.Status = models.OrderStatusNew

	return nil
}

func validateOrder(order *models.Order) error {
	if order.Type != models.OrderTypeMarket {
		return fmt.Errorf("MEXC client supports only market orders")
	}

	if order.Quantity.LessThanOrEqual(decimal.NewFromInt(0)) {
		return fmt.Errorf("MEXC client order quantity must be greater than 0")
	}

	if len(order.Symbol) == 0 {
		return fmt.Errorf("MEXC client order symbol must be specified")
	}

	if order.ID == uuid.Nil {
		return fmt.Errorf("MEXC client order id must be a valid uuid")
	}

	return nil
}
