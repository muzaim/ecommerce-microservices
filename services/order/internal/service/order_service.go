package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"ecommerce/services/order/internal/model"
	"ecommerce/services/order/internal/rabbitmq"
	"ecommerce/services/order/internal/repository"

	"github.com/redis/go-redis/v9"
)

type CachedProductInfo struct {
	ID    uint    `json:"id"`
	Name  string  `json:"name"`
	Price float64 `json:"price"`
	Stock int     `json:"stock"`
}

type OrderService interface {
	CreateOrder(req model.CreateOrderRequest) (*model.Order, error)
	GetOrderByID(id uint) (*model.Order, error)
	GetMyOrders(userID uint) ([]model.Order, error)
}

type orderService struct {
	repo       repository.OrderRepository
	producer   rabbitmq.Producer
	rdb        *redis.Client
	httpClient *http.Client
}

func NewOrderService(repo repository.OrderRepository, producer rabbitmq.Producer, rdb *redis.Client) OrderService {
	return &orderService{
		repo:       repo,
		producer:   producer,
		rdb:        rdb,
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

func (s *orderService) fetchProductFromService(productID uint) (*CachedProductInfo, error) {
	baseURL := os.Getenv("PRODUCT_SERVICE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:8082"
	}

	url := fmt.Sprintf("%s/products/%d", baseURL, productID)
	resp, err := s.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to call product service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("product ID %d not found in product service (status %d)", productID, resp.StatusCode)
	}

	var product CachedProductInfo
	if err := json.NewDecoder(resp.Body).Decode(&product); err != nil {
		return nil, err
	}

	// Lazy Caching: Auto Populate Redis for future requests
	if s.rdb != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		key := fmt.Sprintf("product:%d", productID)
		productJSON, _ := json.Marshal(product)
		s.rdb.Set(ctx, key, string(productJSON), 24*time.Hour)
		log.Printf("[Lazy Caching] Populated Redis cache for product %d from Product Service DB", productID)
	}

	return &product, nil
}

func (s *orderService) CreateOrder(req model.CreateOrderRequest) (*model.Order, error) {
	var totalAmount float64
	var orderItems []model.OrderItem

	for _, itemReq := range req.Items {
		var product *CachedProductInfo

		// 1. Try reading from Redis Cache (0.1ms Fast Validation)
		if s.rdb != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			key := fmt.Sprintf("product:%d", itemReq.ProductID)
			val, err := s.rdb.Get(ctx, key).Result()
			if err == nil && val != "" {
				var cachedProd CachedProductInfo
				if err := json.Unmarshal([]byte(val), &cachedProd); err == nil {
					product = &cachedProd
				}
			}
		}

		// 2. Cache Miss Fallback: Fetch from Product Service DB & Warm Redis Cache
		if product == nil {
			log.Printf("[Order Service] Cache Miss for product %d. Fetching from Product Service...", itemReq.ProductID)
			fetched, err := s.fetchProductFromService(itemReq.ProductID)
			if err != nil {
				return nil, fmt.Errorf("failed to verify product ID %d: %w", itemReq.ProductID, err)
			}
			product = fetched
		}

		// 3. FAST FAIL STOCK VALIDATION: Block HTTP request immediately if stock insufficient!
		if product.Stock < itemReq.Quantity {
			return nil, fmt.Errorf("insufficient stock for product '%s'. Available: %d, Requested: %d",
				product.Name, product.Stock, itemReq.Quantity)
		}

		itemTotal := product.Price * float64(itemReq.Quantity)
		totalAmount += itemTotal

		orderItems = append(orderItems, model.OrderItem{
			ProductID:   itemReq.ProductID,
			ProductName: product.Name,
			Quantity:    itemReq.Quantity,
			Price:       product.Price,
		})
	}

	// 4. Simpan Order di DB lokal dengan status PENDING
	order := model.Order{
		UserID:      req.UserID,
		TotalAmount: totalAmount,
		Status:      "PENDING",
		Items:       orderItems,
	}

	if err := s.repo.CreateOrder(&order); err != nil {
		return nil, err
	}

	// 5. Publish Event OrderCreated ke RabbitMQ untuk Saga Stock Reservation Check
	if s.producer != nil {
		event := map[string]interface{}{
			"event":        "OrderCreated",
			"order_id":     order.ID,
			"user_id":      order.UserID,
			"total_amount": order.TotalAmount,
			"items":        order.Items,
		}
		if err := s.producer.PublishOrderCreated(event); err != nil {
			log.Printf("[Warning] Failed to publish OrderCreated event for Saga: %v", err)
		}
	}

	return &order, nil
}

func (s *orderService) GetOrderByID(id uint) (*model.Order, error) {
	return s.repo.FindOrderByID(id)
}

func (s *orderService) GetMyOrders(userID uint) ([]model.Order, error) {
	return s.repo.FindOrdersByUserID(userID)
}
