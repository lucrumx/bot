package arbitragebot

import (
	"context"

	"gorm.io/gorm"

	"github.com/lucrumx/bot/internal/models"
)

type ArbitrageSpreadRepository interface {
	Create(ctx context.Context, spread *models.ArbitrageSpread) error
	Update(ctx context.Context, spread *models.ArbitrageSpread, where FindFilter) error
	FindAll(ctx context.Context, f FindFilter) ([]*models.ArbitrageSpread, error)
}

type ArbitrageSpreadStatus string

type FindFilter struct {
	Symbol      string
	BuyEx       string
	SellEx      string
	Status      []models.ArbitrageSpreadStatus
	NotInStatus []models.ArbitrageSpreadStatus
}

type GormArbitrageSpreadRepository struct {
	db *gorm.DB
}

func NewGormArbitrageSpreadRepository(db *gorm.DB) *GormArbitrageSpreadRepository {
	return &GormArbitrageSpreadRepository{db: db}
}

func (r *GormArbitrageSpreadRepository) Create(ctx context.Context, spread *models.ArbitrageSpread) error {
	return r.db.WithContext(ctx).Create(spread).Error
}

func (r *GormArbitrageSpreadRepository) Update(ctx context.Context, spread *models.ArbitrageSpread, where FindFilter) error {
	filters := map[string]interface{}{}
	if where.Symbol != "" {
		filters["symbol"] = where.Symbol
	}
	if where.BuyEx != "" {
		filters["buy_on_exchange"] = where.BuyEx
	}
	if where.SellEx != "" {
		filters["sell_on_exchange"] = where.SellEx
	}
	if len(where.Status) > 0 {
		filters["status"] = gorm.Expr("IN (?)", where.Status)
	}
	if len(where.NotInStatus) > 0 {
		filters["status"] = gorm.Expr("NOT IN (?)", where.NotInStatus)
	}

	return r.db.WithContext(ctx).Model(&models.ArbitrageSpread{}).Where(filters).Updates(spread).Error
}

func (r *GormArbitrageSpreadRepository) FindAll(
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
