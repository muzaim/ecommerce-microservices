package main

import (
	"log"
	"net/http"
	"os"

	"ecommerce/services/order/internal/config"
	"ecommerce/services/order/internal/handler"
	"ecommerce/services/order/internal/rabbitmq"
	"ecommerce/services/order/internal/repository"
	"ecommerce/services/order/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	db := config.ConnectDB()
	rdb := config.ConnectRedis()

	orderRepo := repository.NewOrderRepository(db)

	rabbitmqURL := os.Getenv("RABBITMQ_URL")

	producer, err := rabbitmq.NewProducer(rabbitmqURL)
	if err != nil {
		log.Printf("[Warning] Failed to connect Producer to RabbitMQ: %v", err)
	} else {
		defer producer.Close()
	}

	rabbitmq.StartConsumer(rabbitmqURL, orderRepo)

	orderService := service.NewOrderService(orderRepo, producer, rdb)
	orderHandler := handler.NewOrderHandler(orderService)

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "OK",
			"service": "order-service",
		})
	})

	orderRoutes := r.Group("/orders")
	{
		orderRoutes.POST("", orderHandler.Create)
		orderRoutes.GET("/my-orders", orderHandler.GetMyOrders)
		orderRoutes.GET("/:id", orderHandler.GetByID)
	}

	log.Println("Order service running on port :8083")
	if err := r.Run(":8083"); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
