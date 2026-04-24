package arbitragebot

import (
	"context"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"

	"github.com/lucrumx/bot/internal/models"
)

// OrderRepository persists exchange orders.
//
//mockery:generate: true
type OrderRepository interface {
	Create(ctx context.Context, order *models.Order) error
	UpdateFilled(ctx context.Context, id uuid.UUID, avgPrice decimal.Decimal, executedQty decimal.Decimal) error
}

// orderRepository is a GORM implementation of OrderRepository.
type orderRepository struct {
	db *gorm.DB
}

// NewOrderRepository creates a new orderRepository.
func NewOrderRepository(db *gorm.DB) OrderRepository {
	return &orderRepository{db: db}
}

// Create inserts a new order record.
func (r *orderRepository) Create(ctx context.Context, order *models.Order) error {
	return r.db.WithContext(ctx).Create(order).Error
}

// UpdateFilled updates the order to filled status with execution details.
func (r *orderRepository) UpdateFilled(ctx context.Context, id uuid.UUID, avgPrice decimal.Decimal, executedQty decimal.Decimal) error {
	return r.db.WithContext(ctx).
		Model(&models.Order{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":            models.OrderStatusFilled,
			"avg_price":         avgPrice,
			"executed_quantity": executedQty,
		}).Error
}
