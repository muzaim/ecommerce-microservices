package repository

import (
	"ecommerce/services/order/internal/model"
	"gorm.io/gorm"
)

type OrderRepository interface {
	CreateOrder(order *model.Order) error
	FindOrderByID(id uint) (*model.Order, error)
	FindOrdersByUserID(userID uint) ([]model.Order, error)
	UpdateOrderStatus(orderID uint, status string) error
}

type orderRepository struct {
	db *gorm.DB
}

func NewOrderRepository(db *gorm.DB) OrderRepository {
	return &orderRepository{db: db}
}

func (r *orderRepository) CreateOrder(order *model.Order) error {
	return r.db.Create(order).Error
}

func (r *orderRepository) FindOrderByID(id uint) (*model.Order, error) {
	var order model.Order
	err := r.db.Preload("Items").First(&order, id).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func (r *orderRepository) FindOrdersByUserID(userID uint) ([]model.Order, error) {
	var orders []model.Order
	err := r.db.Preload("Items").Where("user_id = ?", userID).Order("id desc").Find(&orders).Error
	return orders, err
}

func (r *orderRepository) UpdateOrderStatus(orderID uint, status string) error {
	return r.db.Model(&model.Order{}).Where("id = ?", orderID).Update("status", status).Error
}
