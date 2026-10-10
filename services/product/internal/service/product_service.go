package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"ecommerce/services/product/internal/model"
	"ecommerce/services/product/internal/rabbitmq"
	"ecommerce/services/product/internal/repository"

	"github.com/redis/go-redis/v9"
)

type ProductService interface {
	CreateProduct(req model.CreateProductRequest) (*model.Product, error)
	GetAllProducts() ([]model.Product, error)
	GetProductByID(id uint) (*model.Product, error)
}

type productService struct {
	repo     repository.ProductRepository
	producer rabbitmq.Producer
	rdb      *redis.Client
}

func NewProductService(repo repository.ProductRepository, producer rabbitmq.Producer, rdb *redis.Client) ProductService {
	return &productService{
		repo:     repo,
		producer: producer,
		rdb:      rdb,
	}
}

func (s *productService) CreateProduct(req model.CreateProductRequest) (*model.Product, error) {
	product := model.Product{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Stock:       req.Stock,
	}
	if err := s.repo.Create(&product); err != nil {
		return nil, err
	}

	// 1. Write Product Cache to Redis (Global Product Cache)
	if s.rdb != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		productJSON, err := json.Marshal(product)
		if err == nil {
			key := fmt.Sprintf("product:%d", product.ID)
			if err := s.rdb.Set(ctx, key, string(productJSON), 24*time.Hour).Err(); err != nil {
				log.Printf("[Redis Warning] Failed to cache product %d: %v", product.ID, err)
			} else {
				log.Printf("[Redis Success] Cached product %d in Redis: %s", product.ID, product.Name)
			}
		}
	}

	// 2. Publish Event ProductCreated ke RabbitMQ secara asynchronous
	event := map[string]interface{}{
		"event":       "ProductCreated",
		"id":          product.ID,
		"name":        product.Name,
		"price":       product.Price,
		"stock":       product.Stock,
		"description": product.Description,
	}
	if err := s.producer.PublishProductCreated(event); err != nil {
		log.Printf("[Warning] Failed to publish ProductCreated event: %v", err)
	}

	return &product, nil
}

func (s *productService) GetAllProducts() ([]model.Product, error) {
	return s.repo.FindAll()
}

func (s *productService) GetProductByID(id uint) (*model.Product, error) {
	// Try reading from Redis Cache first
	if s.rdb != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		key := fmt.Sprintf("product:%d", id)
		val, err := s.rdb.Get(ctx, key).Result()
		if err == nil && val != "" {
			var product model.Product
			if err := json.Unmarshal([]byte(val), &product); err == nil {
				return &product, nil
			}
		}
	}

	// Cache Miss: Query Database
	product, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}

	// Cache in Redis for future reads
	if s.rdb != nil && product != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		productJSON, _ := json.Marshal(product)
		key := fmt.Sprintf("product:%d", product.ID)
		s.rdb.Set(ctx, key, string(productJSON), 24*time.Hour)
	}

	return product, nil
}
