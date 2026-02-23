package bybit

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/lucrumx/bot/internal/exchange/client/bybit/dtos"
	"github.com/lucrumx/bot/internal/models"
)

// Api description: https://bybit-exchange.github.io/docs/v5/order/create-order

const orderURL = "/v5/order/create"

// CreateOrder creates an order on the exchange.
func (c *Client) CreateOrder(ctx context.Context, order models.Order) error {
	if err := validateBeforeCreateOrder(&order); err != nil {
		return err
	}

	apiKey := c.cfg.Exchange.ByBit.APIKey
	recvWindow := strconv.FormatInt(c.cfg.Exchange.ByBit.RecvWindow, 10)

	payload := mapRequestDataToOrderDTO(&order)
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("ByBit client failed to marshal create order request: %w", err)
	}
	bodyStr := string(bodyBytes)
	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	signStr := timestamp + apiKey + recvWindow + bodyStr

	h := hmac.New(sha256.New, []byte(c.cfg.Exchange.ByBit.APISecret))
	h.Write([]byte(signStr))
	signature := hex.EncodeToString(h.Sum(nil))

	fmt.Printf("Sign string: %s\n", signStr)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+orderURL,
		bytes.NewBuffer(bodyBytes))

	if err != nil {
		return fmt.Errorf("ByBit client failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-BAPI-API-KEY", apiKey)
	req.Header.Set("X-BAPI-TIMESTAMP", timestamp)
	req.Header.Set("X-BAPI-RECV-WINDOW", recvWindow)
	req.Header.Set("X-BAPI-SIGN", signature)

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("ByBit client http create order request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("ByBit client failed to read create order response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ByBit client unexpected http while creating order, status code: %d, body: %s", resp.StatusCode, string(body))
	}

	var raw dtos.OrderCreateResponseDTO

	if err := json.Unmarshal(body, &raw); err != nil {
		return fmt.Errorf("ByBit client failed to unmarshal create order response: %w", err)
	}

	if raw.RetCode != 0 {
		return fmt.Errorf("ByBit client failed to create order, code: %d, msg: %s", raw.RetCode, raw.RetMsg)
	}

	return nil
}

func validateBeforeCreateOrder(order *models.Order) error {
	if order.Market != models.OrderMarketLinear {
		return fmt.Errorf("ByBit client linear (category) market")
	}

	if order.Type != models.OrderTypeMarket {
		return fmt.Errorf("ByBit client support only market orders")
	}

	if order.Quantity.LessThanOrEqual(decimal.NewFromInt(0)) {
		return fmt.Errorf("ByBit client order quantity must be greater than 0")
	}

	if len(order.Symbol) <= 0 {
		return fmt.Errorf("ByBit client order symbol must be specified")
	}

	if order.ID == uuid.Nil {
		return fmt.Errorf("ByBit client order id must be valid uuid")
	}

	return nil
}

func mapRequestDataToOrderDTO(order *models.Order) map[string]interface{} {
	side := dtos.OrderSideBuy

	if order.Side == models.OrderSideSell {
		side = dtos.OrderSideSell
	}
	orderType := dtos.OrderTypeMarket
	if order.Type == models.OrderTypeLimit {
		orderType = dtos.OrderTypeLimit
	}

	market := "linear"
	if order.Market == models.OrderMarketSpot {
		market = "spot"
	}

	return map[string]interface{}{
		"category":    market,
		"symbol":      order.Symbol,
		"side":        string(side),
		"orderType":   string(orderType),
		"qty":         order.Quantity.String(),
		"orderLinkId": order.ID.String(),
		//"isLeverage":  0, // если спот и ордер за счет маржи (заемных средств) - должен быть = 1
	}
}
