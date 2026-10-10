package repository

import (
	"fmt"

	"ecommerce/services/product/internal/model"

	"gorm.io/gorm"
)

type ProductRepository interface {
	Create(product *model.Product) error
	Update(product *model.Product) error
	FindAll() ([]model.Product, error)
	FindByID(id uint) (*model.Product, error)
	DeductStock(productID uint, quantity int) (*model.Product, error)
}

type productRepository struct {
	db *gorm.DB
}

func NewProductRepository(db *gorm.DB) ProductRepository {
	return &productRepository{
		db: db,
	}
}

func (r *productRepository) Create(product *model.Product) error {
	return r.db.Create(product).Error
}

func (r *productRepository) Update(product *model.Product) error {
	return r.db.Save(product).Error
}

func (r *productRepository) FindAll() ([]model.Product, error) {
	var products []model.Product
	err := r.db.Find(&products).Error
	return products, err
}

func (r *productRepository) FindByID(id uint) (*model.Product, error) {
	var product model.Product
	err := r.db.First(&product, id).Error
	return &product, err
}

func (r *productRepository) DeductStock(productID uint, quantity int) (*model.Product, error) {
	var updatedProduct model.Product
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var product model.Product
		if err := tx.Set("gorm:query_option", "FOR UPDATE").First(&product, productID).Error; err != nil {
			return fmt.Errorf("product ID %d not found", productID)
		}

		if product.Stock < quantity {
			return fmt.Errorf("insufficient stock for product ID %d. Available: %d, Requested: %d", productID, product.Stock, quantity)
		}

		newStock := product.Stock - quantity
		if err := tx.Model(&product).Update("stock", newStock).Error; err != nil {
			return err
		}

		product.Stock = newStock
		updatedProduct = product
		return nil
	})

	if err != nil {
		return nil, err
	}
	return &updatedProduct, nil
}
