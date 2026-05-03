package arbitragebot

import (
	"context"

	"gorm.io/gorm"

	"github.com/google/uuid"

	"github.com/lucrumx/bot/internal/models"
)

// ArbitrageSpreadRepository represents a db repository for arbitrage spreads.
type ArbitrageSpreadRepository interface {
	Create(ctx context.Context, spread *models.ArbitrageSpread) error
	Update(ctx context.Context, spread *models.ArbitrageSpread, where FindFilter) error
	FindAll(ctx context.Context, f FindFilter) ([]*models.ArbitrageSpread, error)
	FindOne(ctx context.Context, f FindFilter) (*models.ArbitrageSpread, error)
	FindOneByOrderID(ctx context.Context, orderID uuid.UUID) (*models.ArbitrageSpread, error)
}

// FindFilter is a filter for finding arbitrage spreads in the repository.
type FindFilter struct {
	Symbol      string
	BuyEx       string
	SellEx      string
	Status      []models.ArbitrageSpreadStatus
	NotInStatus []models.ArbitrageSpreadStatus
	ID          uuid.UUID
}

// Repository is a GORM implementation of ArbitrageSpreadRepository interface.
type Repository struct {
	db *gorm.DB
}

// NewArbitrageSpreadRepository creates a new GormArbitrageSpreadRepository.
func NewArbitrageSpreadRepository(db *gorm.DB) *Repository {
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
	if f.ID != uuid.Nil {
		tx = tx.Where("id = ?", f.ID)
	}

	return tx.Updates(spread).Error
}

func combineFilters(db *gorm.DB, f FindFilter) *gorm.DB {
	if f.Symbol != "" {
		db = db.Where("symbol = ?", f.Symbol)
	}
	if f.BuyEx != "" {
		db = db.Where("buy_on_exchange = ?", f.BuyEx)
	}
	if f.SellEx != "" {
		db = db.Where("sell_on_exchange = ?", f.SellEx)
	}
	if len(f.Status) > 0 {
		db = db.Where("status IN ?", f.Status) // GORM сам поймет слайс
	}
	if len(f.NotInStatus) > 0 {
		db = db.Where("status NOT IN ?", f.NotInStatus)
	}

	return db
}

// FindAll finds all arbitrage spreads in the repository based on the provided filter.
func (r *Repository) FindAll(
	ctx context.Context,
	f FindFilter,
) ([]*models.ArbitrageSpread, error) {

	var spreads []*models.ArbitrageSpread

	tx := r.db.WithContext(ctx).Model(&models.ArbitrageSpread{})
	tx = combineFilters(tx, f)

	result := tx.Find(&spreads)
	if result.Error != nil {
		return nil, result.Error
	}

	return spreads, nil
}

// FindOne finds first arbitrage spreads in the repository based on the provided filter.
func (r *Repository) FindOne(
	ctx context.Context,
	f FindFilter,
) (*models.ArbitrageSpread, error) {

	var spread *models.ArbitrageSpread
	tx := r.db.WithContext(ctx).Model(&models.ArbitrageSpread{})
	tx = combineFilters(tx, f)

	result := tx.First(&spread)
	if result.Error != nil {
		return nil, result.Error
	}

	return spread, nil
}

// FindOneByOrderID finds a single arbitrage spread by its associated order ID.
func (r *Repository) FindOneByOrderID(ctx context.Context, orderID uuid.UUID) (*models.ArbitrageSpread, error) {
	var spread models.ArbitrageSpread
	result := r.db.WithContext(ctx).
		Where("open_buy_order_id = ?", orderID).
		Or("open_sell_order_id = ?", orderID).
		Or("close_buy_order_id = ?", orderID).
		Or("close_sell_order_id = ?", orderID).
		First(&spread)

	if result.Error != nil {
		return nil, result.Error
	}
	return &spread, nil
}
