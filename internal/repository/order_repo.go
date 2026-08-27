package repository

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"tixora/internal/models"
)

type IOrderRepository interface {
	Create(ctx context.Context, order *models.Order) error
	GetByID(ctx context.Context, id string) (*models.Order, error)
	GetByOrderID(ctx context.Context, orderID string) (*models.Order, error)
	GetByUserID(ctx context.Context, userID string, status models.OrderStatus, offset, limit int) ([]models.Order, int64, error)
	GetWithPagination(ctx context.Context, status models.OrderStatus, offset, limit int) ([]models.Order, int64, error)
	GetUserOrderStats(ctx context.Context, userID string) (totalOrders int64, totalSpent int64, err error)
	Update(ctx context.Context, order *models.Order) error
	UpdateStatus(ctx context.Context, id string, status models.OrderStatus) error
}

type OrderRepository struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) IOrderRepository {
	return &OrderRepository{db: db}
}

func (r *OrderRepository) Create(ctx context.Context, order *models.Order) error {
	if err := r.db.WithContext(ctx).Create(order).Error; err != nil {
		return fmt.Errorf("failed to create order: %w", err)
	}
	return nil
}

func (r *OrderRepository) GetByID(ctx context.Context, id string) (*models.Order, error) {
	var order models.Order
	if err := r.db.WithContext(ctx).Where("id = ?", id).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get order by id: %w", err)
	}
	return &order, nil
}

func (r *OrderRepository) GetByOrderID(ctx context.Context, orderID string) (*models.Order, error) {
	var order models.Order
	if err := r.db.WithContext(ctx).
		Preload("Event").
		Preload("Event.Image").
		Preload("User").
		Where("order_id = ?", orderID).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get order by order id: %w", err)
	}
	return &order, nil
}

func (r *OrderRepository) GetByUserID(ctx context.Context, userID string, status models.OrderStatus, offset, limit int) ([]models.Order, int64, error) {
	var orders []models.Order
	var total int64

	countQuery := r.db.WithContext(ctx).Model(&models.Order{}).Where("user_id = ?", userID)
	findQuery := r.db.WithContext(ctx).Where("user_id = ?", userID)
	if status != "" {
		countQuery = countQuery.Where("status = ?", status)
		findQuery = findQuery.Where("status = ?", status)
	}

	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count orders for user: %w", err)
	}

	if err := findQuery.
		Preload("Event").
		Preload("Event.Image").
		Offset(offset).
		Limit(limit).
		Order("created_at desc").
		Find(&orders).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to fetch orders for user: %w", err)
	}

	return orders, total, nil
}

func (r *OrderRepository) GetWithPagination(ctx context.Context, status models.OrderStatus, offset, limit int) ([]models.Order, int64, error) {
	var orders []models.Order
	var total int64

	countQuery := r.db.WithContext(ctx).Model(&models.Order{})
	findQuery := r.db.WithContext(ctx)
	if status != "" {
		countQuery = countQuery.Where("status = ?", status)
		findQuery = findQuery.Where("status = ?", status)
	}

	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count orders: %w", err)
	}

	if err := findQuery.
		Preload("Event").
		Preload("Event.Image").
		Preload("User").
		Offset(offset).
		Limit(limit).
		Order("created_at desc").
		Find(&orders).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to fetch orders: %w", err)
	}

	return orders, total, nil
}

func (r *OrderRepository) GetUserOrderStats(ctx context.Context, userID string) (int64, int64, error) {
	var result struct {
		TotalOrders int64
		TotalSpent  int64
	}

	if err := r.db.WithContext(ctx).Model(&models.Order{}).
		Where("user_id = ? AND status = ?", userID, models.OrderStatusPaid).
		Select("COUNT(*) AS total_orders, COALESCE(SUM(total_price), 0) AS total_spent").
		Scan(&result).Error; err != nil {
		return 0, 0, fmt.Errorf("failed to fetch order stats for user: %w", err)
	}

	return result.TotalOrders, result.TotalSpent, nil
}

func (r *OrderRepository) Update(ctx context.Context, order *models.Order) error {
	if err := r.db.WithContext(ctx).Save(order).Error; err != nil {
		return fmt.Errorf("failed to update order: %w", err)
	}
	return nil
}

func (r *OrderRepository) UpdateStatus(ctx context.Context, id string, status models.OrderStatus) error {
	result := r.db.WithContext(ctx).Model(&models.Order{}).
		Where("id = ?", id).
		Update("status", status)
	if result.Error != nil {
		return fmt.Errorf("failed to update order status: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("update order status: %w", gorm.ErrRecordNotFound)
	}
	return nil
}
