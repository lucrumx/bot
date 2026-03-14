package bybit

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/shopspring/decimal"

	"github.com/lucrumx/bot/internal/exchange/client/bybit/dtos"
	"github.com/lucrumx/bot/internal/models"
)

const balanceURL = "/v5/account/wallet-balance"
const accountType = "UNIFIED" // единый торговый аккаунт

// GetBalances returns the balances of the user's account on the exchange.
func (c *Client) GetBalances(ctx context.Context) ([]models.Balance, error) {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL+balanceURL,
		nil,
	)
	if err != nil {
		return nil, fmt.Errorf("ByBit | GetBalance Method: client failed to create request: %w", err)
	}

	query := req.URL.Query()
	query.Set("accountType", accountType)
	req.URL.RawQuery = query.Encode()

	timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
	recvWindow := "5000"
	payload := timestamp + c.cfg.Exchange.ByBit.APIKey + recvWindow + req.URL.RawQuery

	signature := sign(c.cfg.Exchange.ByBit.APISecret, payload)

	c.setHeader(req, signature, timestamp, recvWindow)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ByBit | GetBalance Method: client http get balances request failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ByBit | GetBalance Method: client failed to read get balances response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"ByBit | GetBalance Method: unexpected http status %d: %s",
			resp.StatusCode,
			string(body),
		)
	}

	var raw dtos.Balance
	if err = json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("ByBit | GetBalance Method: client failed to unmarshal get balances response: %w", err)
	}

	if raw.RetCode != 0 {
		return nil, &apiError{Code: raw.RetCode, Message: raw.RetMsg}
	}

	result, err := c.mapper(raw)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (c *Client) mapper(raw dtos.Balance) ([]models.Balance, error) {
	result := make([]models.Balance, 0, len(raw.Result.List))
	for _, l := range raw.Result.List {
		if l.AccountType != accountType {
			continue
		}

		for _, coin := range l.Coin {
			walletBalance, err := decimal.NewFromString(coin.WalletBalance)
			if err != nil {
				return nil, fmt.Errorf("ByBit | GetBalance mapper: client failed to parse wallet balance: %w", err)
			}
			locked, err := decimal.NewFromString(coin.Locked)
			if err != nil {
				return nil, fmt.Errorf("ByBit | GetBalance mapper: client failed to parse locked: %w", err)
			}

			result = append(result, models.Balance{
				ExchangeName: c.GetExchangeName(),
				Asset:        coin.Coin,
				Free:         walletBalance.Sub(locked),
				Locked:       locked,
				Total:        walletBalance,
			})
		}
	}

	return result, nil
}
