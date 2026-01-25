// Package bybit provides a client for the ByBit exchange.
package bybit

import (
	"net/http"

	"github.com/lucrumx/bot/internal/utils"
)

// Client represents a ByBit client.
type Client struct {
	baseURL string
	http    *http.Client
}

// NewByBitClient creates a new ByBitClient.
func NewByBitClient() *Client {
	return &Client{
		baseURL: utils.GetEnv("BYBIT_BASE_URL", ""),
		http:    &http.Client{},
	}
}
