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
	GetByID(ctx context.Context, id uuid.UUID) (*models.Order, error)
	Create(ctx context.Context, order *models.Order) error
	UpdateFilled(ctx context.Context, id uuid.UUID, avgPrice decimal.Decimal, executedQty decimal.Decimal) error
	UpdatePartialy(ctx context.Context, id uuid.UUID, patch OrderPatch) error
}

// orderRepository is a GORM implementation of OrderRepository.
type orderRepository struct {
	db *gorm.DB
}

// NewOrderRepository creates a new orderRepository.
func NewOrderRepository(db *gorm.DB) OrderRepository {
	return &orderRepository{db: db}
}

func (r *orderRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Order, error) {
	var order models.Order
	if err := r.db.WithContext(ctx).First(&order, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &order, nil
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

// OrderPatch represents the fields that can be updated for an order after execution.
type OrderPatch struct {
	Status           *models.OrderStatus
	AvgPrice         *decimal.Decimal
	ExecutedQuantity *decimal.Decimal
	Fees             *decimal.Decimal
	Profit           *decimal.Decimal
	HasErrors        *bool
}

func (r *orderRepository) UpdatePartialy(ctx context.Context, id uuid.UUID, patch OrderPatch) error {
	return r.db.WithContext(ctx).
		Model(&models.Order{}).
		Where("id = ?", id).
		Updates(patch).Error
}
