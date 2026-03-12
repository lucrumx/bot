package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// Balance represents the available, locked, and total balances for a specific asset on an exchange.
type Balance struct {
	ID           uuid.UUID       `gorm:"type:uuid;primary_key;default:uuidv7()"`
	ExchangeName string          `gorm:"type:text;not null;uniqueIndex:idx_balance_exchange_asset"`
	Asset        string          `gorm:"type:text;not null;uniqueIndex:idx_balance_exchange_asset"`
	Free         decimal.Decimal `gorm:"type:decimal(28,12);"`
	Locked       decimal.Decimal `gorm:"type:decimal(28,12);"`
	Total        decimal.Decimal `gorm:"type:decimal(28,12);"`
	CreatedAt    time.Time       `gorm:"type:timestamptz;default:now()"`
	UpdatedAt    time.Time       `gorm:"type:timestamptz;"`
}

/** маппинг для bingx
r := models.Balance{
			ExchangeName: exchangeName,
			Asset:        data.Asset,
			Free:         decimal.NewFromFloat(float64(data.AvailableMargin)),
			Locked:       decimal.NewFromFloat(float64(data.FreezedMargin)),
			Total:        decimal.NewFromFloat(float64(data.Equity)),
		}
*/
