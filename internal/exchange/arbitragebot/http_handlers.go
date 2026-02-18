package arbitragebot

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

// HTTPHandlers contains HTTP handlers for the arbitrage bot.
type HTTPHandlers struct {
	repo ArbitrageSpreadRepository
}

// NewHTTPHandlers creates a new instance of HttpHandlers with the provided repository.
func NewHTTPHandlers(repo ArbitrageSpreadRepository) *HTTPHandlers {
	return &HTTPHandlers{
		repo: repo,
	}
}

// GetSpreadsHandler handles the HTTP request to get all arbitrage spreads.
func (h *HTTPHandlers) GetSpreadsHandler(c *gin.Context) {
	spreads, err := h.repo.FindAll(c.Request.Context(), FindFilter{})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	type spreadResponse struct {
		ID               string          `json:"id"`
		CreatedAt        time.Time       `json:"created_at"`
		UpdatedAt        time.Time       `json:"updated_at"`
		Symbol           string          `json:"symbol"`
		BuyOnExchange    string          `json:"buy_on_exchange"`
		SellOnExchange   string          `json:"sell_on_exchange"`
		BuyPrice         decimal.Decimal `json:"buy_price"`
		SellPrice        decimal.Decimal `json:"sell_price"`
		SpreadPercent    decimal.Decimal `json:"spread_percent"`
		MaxSpreadPercent decimal.Decimal `json:"max_spread_percent"`
		Status           string          `json:"status"`
	}

	var response []spreadResponse
	for _, spread := range spreads {
		response = append(response, spreadResponse{
			ID:               spread.ID.String(),
			CreatedAt:        spread.CreatedAt,
			UpdatedAt:        spread.UpdatedAt,
			Symbol:           spread.Symbol,
			BuyOnExchange:    spread.BuyOnExchange,
			SellOnExchange:   spread.SellOnExchange,
			BuyPrice:         spread.BuyPrice,
			SellPrice:        spread.SellPrice,
			SpreadPercent:    spread.SpreadPercent,
			MaxSpreadPercent: spread.MaxSpreadPercent,
			Status:           string(spread.Status),
		})
	}

	c.JSON(http.StatusOK, response)
}
