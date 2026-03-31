package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ArbitrageTradeStatus represents the status of an arbitrage trade.
type ArbitrageTradeStatus string

const (
	// ArbitrageTradeStatusNew NEW, запись создана, заявки ещё не отправлены
	ArbitrageTradeStatusNew ArbitrageTradeStatus = "NEW"

	// ArbitrageTradeStatusOpening OPENING бот отправил заявки на открытие
	ArbitrageTradeStatusOpening ArbitrageTradeStatus = "OPENING"

	// ArbitrageTradeStatusOpenPartial OPEN_PARTIAL одна нога открылась, вторая ещё нет или уже сломалась
	ArbitrageTradeStatusOpenPartial ArbitrageTradeStatus = "OPEN_PARTIAL"

	// ArbitrageTradeStatusOpened OPENED обе ноги открыты
	ArbitrageTradeStatusOpened ArbitrageTradeStatus = "OPENED"

	// ArbitrageTradeStatusClosing CLOSING отправлены заявки на закрытие
	ArbitrageTradeStatusClosing ArbitrageTradeStatus = "CLOSING"

	// ArbitrageTradeStatusClosePartial CLOSE_PARTIAL одна нога закрылась, вторая нет
	ArbitrageTradeStatusClosePartial ArbitrageTradeStatus = "CLOSE_PARTIAL"

	// ArbitrageTradeStatusClosed CLOSED обе ноги закрыты
	ArbitrageTradeStatusClosed ArbitrageTradeStatus = "CLOSED"

	// ArbitrageTradeStatusFailed FAILED сделка не смогла открыться
	ArbitrageTradeStatusFailed ArbitrageTradeStatus = "FAILED"

	// ArbitrageTradeStatusRecoveryRequired RECOVERY_REQUIRED см руками, что-то сломалось
	ArbitrageTradeStatusRecoveryRequired ArbitrageTradeStatus = "RECOVERY_REQUIRED"
)

// ArbitrageTrade арбитражная сделка
type ArbitrageTrade struct {
	ID                 uuid.UUID            `gorm:"type:uuid;primary_key;default:uuidv7()"`
	SpreadKey          string               `gorm:"type:text;not null;"` // symbol + buyExchange + sellExchange
	Symbol             string               `gorm:"type:text;not null;"`
	BuyExchange        string               `gorm:"type:text;not null;"`
	SellExchange       string               `gorm:"type:text;not null;"`
	Status             ArbitrageTradeStatus `gorm:"type:text;not null;"`
	OpenSignalAt       time.Time            `gorm:"type:timestamptz;"`
	CloseSignalAt      time.Time            `gorm:"type:timestamptz;"`
	OpenedAt           time.Time            `gorm:"type:timestamptz;"`
	ClosedAt           time.Time            `gorm:"type:timestamptz;"`
	TargetMarginUSDT   decimal.Decimal      `gorm:"type:decimal(28,12);"` // свои
	TargetNotionalUSDT decimal.Decimal      `gorm:"type:decimal(28,12);"` // с учетом плеча
	OpenBuyOrderID     uuid.UUID            `gorm:"type:uuid;"`
	OpenSellOrderID    string               `gorm:"type:uuid;"`
	CloseBuyOrderID    string               `gorm:"type:uuid;"`
	CloseSellOrderID   string               `gorm:"type:uuid"`
	LastError          string               `gorm:"type:text;"`
	CreatedAt          time.Time            `gorm:"type:timestamptz;default:now()"`
	UpdatedAt          time.Time            `gorm:"type:timestamptz;"`
}
