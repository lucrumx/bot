package arbitragebot

import (
	"context"

	"gorm.io/gorm"

	"github.com/lucrumx/bot/internal/models"
)

// ArbitrageSpreadRepository represents a db repository for arbitrage spreads.
type ArbitrageSpreadRepository interface {
	Create(ctx context.Context, spread *models.ArbitrageSpread) error
	Update(ctx context.Context, spread *models.ArbitrageSpread, where FindFilter) error
	FindAll(ctx context.Context, f FindFilter) ([]*models.ArbitrageSpread, error)
}

// FindFilter is a filter for finding arbitrage spreads in the repository.
type FindFilter struct {
	Symbol      string
	BuyEx       string
	SellEx      string
	Status      []models.ArbitrageSpreadStatus
	NotInStatus []models.ArbitrageSpreadStatus
}

// Repository is a GORM implementation of ArbitrageSpreadRepository.
type Repository struct {
	db *gorm.DB
}

// NewRepository creates a new GormArbitrageSpreadRepository.
func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

// Create creates a new arbitrage spread in the repository.
func (r *Repository) Create(ctx context.Context, spread *models.ArbitrageSpread) error {
	return r.db.WithContext(ctx).Create(spread).Error
}

// Update updates an existing arbitrage spread in the repository based on the provided filter.
func (r *Repository) Update(ctx context.Context, spread *models.ArbitrageSpread, f FindFilter) error {
	tx := r.db.WithContext(ctx).Model(&models.ArbitrageSpread{})

	if f.Symbol != "" {
		tx = tx.Where("symbol = ?", f.Symbol)
	}
	if f.BuyEx != "" {
		tx = tx.Where("buy_on_exchange = ?", f.BuyEx)
	}
	if f.SellEx != "" {
		tx = tx.Where("sell_on_exchange = ?", f.SellEx)
	}
	if len(f.Status) > 0 {
		tx = tx.Where("status IN ?", f.Status) // GORM сам поймет слайс
	}
	if len(f.NotInStatus) > 0 {
		tx = tx.Where("status NOT IN ?", f.NotInStatus)
	}

	return tx.Updates(spread).Error
}

// FindAll finds all arbitrage spreads in the repository based on the provided filter.
func (r *Repository) FindAll(
	ctx context.Context,
	f FindFilter,
) ([]*models.ArbitrageSpread, error) {

	var spreads []*models.ArbitrageSpread
	filters := map[string]interface{}{}
	if f.Symbol != "" {
		filters["symbol"] = f.Symbol
	}
	if f.BuyEx != "" {
		filters["buy_on_exchange"] = f.BuyEx
	}
	if f.SellEx != "" {
		filters["sell_on_exchange"] = f.SellEx
	}
	if len(f.Status) > 0 {
		filters["status"] = gorm.Expr("IN (?)", f.Status)
	}
	if len(f.NotInStatus) > 0 {
		filters["status"] = gorm.Expr("NOT IN (?)", f.NotInStatus)
	}

	tx := r.db.WithContext(ctx).Model(&models.ArbitrageSpread{})
	if len(filters) > 0 {
		tx = tx.Where(filters)
	}

	result := tx.Find(&spreads)
	if result.Error != nil {
		return nil, result.Error
	}

	return spreads, nil
}
