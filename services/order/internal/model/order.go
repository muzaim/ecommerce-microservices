package model

import "time"

type Order struct {
	ID          uint        `gorm:"primaryKey" json:"id"`
	UserID      uint        `gorm:"not null" json:"user_id"`
	TotalAmount float64     `gorm:"type:numeric(12,2);not null" json:"total_amount"`
	Status      string      `gorm:"not null;default:'PENDING'" json:"status"`
	Items       []OrderItem `gorm:"foreignKey:OrderID" json:"items"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

type OrderItem struct {
	ID          uint    `gorm:"primaryKey" json:"id"`
	OrderID     uint    `gorm:"not null" json:"order_id"`
	ProductID   uint    `gorm:"not null" json:"product_id"`
	ProductName string  `gorm:"column:product_name;not null" json:"product_name"`
	Quantity    int     `gorm:"not null" json:"quantity"`
	Price       float64 `gorm:"type:numeric(12,2);not null" json:"price"`
}

type CreateOrderItemRequest struct {
	ProductID uint `json:"product_id" binding:"required"`
	Quantity  int  `json:"quantity" binding:"required,gt=0"`
}

type CreateOrderRequest struct {
	UserID uint                     `json:"user_id"`
	Items  []CreateOrderItemRequest `json:"items" binding:"required,min=1"`
}
