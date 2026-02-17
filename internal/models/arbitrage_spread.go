package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// ArbitrageSpreadStatus represents the status of an arbitrage spread.
type ArbitrageSpreadStatus string

const (
	ArbitrageSpreadOpened  ArbitrageSpreadStatus = "OPENED"
	ArbitrageSpreadClosed  ArbitrageSpreadStatus = "CLOSED"
	ArbitrageSpreadUpdated ArbitrageSpreadStatus = "UPDATED" // опционально
)

// ArbitrageSpread represents a spread between two exchanges.
type ArbitrageSpread struct {
	ID uuid.UUID `gorm:"type:uuid;default:uuidv7();primaryKey"`

	CreatedAt time.Time `gorm:"index:idx_spread_created_at;default:now()"`
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`

	Symbol         string `gorm:"index:idx_spread_symbol_exchanges"`
	BuyOnExchange  string `gorm:"index:idx_spread_symbol_exchanges"`
	SellOnExchange string `gorm:"index:idx_spread_symbol_exchanges"`

	BuyPrice         decimal.Decimal       `gorm:"type:decimal(28,12);not null"`
	SellPrice        decimal.Decimal       `gorm:"type:decimal(28,12);not null"`
	SpreadPercent    decimal.Decimal       `gorm:"type:decimal(10,4);not null"`
	MaxSpreadPercent decimal.Decimal       `gorm:"type:decimal(10,4);not null"`
	Status           ArbitrageSpreadStatus `gorm:"type:varchar(20);not null"`
}
