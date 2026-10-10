package main

import (
	"log"
	"net/http"
	"os"

	"ecommerce/services/product/internal/config"
	"ecommerce/services/product/internal/handler"
	"ecommerce/services/product/internal/rabbitmq"
	"ecommerce/services/product/internal/repository"
	"ecommerce/services/product/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	db := config.ConnectDB()
	rdb := config.ConnectRedis()

	rabbitmqURL := os.Getenv("RABBITMQ_URL")
	producer, err := rabbitmq.NewProducer(rabbitmqURL)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer producer.Close()

	productRepo := repository.NewProductRepository(db)

	// Start Background Consumer dengan Real-Time Redis Cache Invalidation/Update
	rabbitmq.StartConsumer(rabbitmqURL, productRepo, producer, rdb)

	productService := service.NewProductService(productRepo, producer, rdb)
	productHandler := handler.NewProductHandler(productService)

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "OK",
			"service": "product-service",
		})
	})

	productRoutes := r.Group("/products")
	{
		productRoutes.POST("", productHandler.Create)
		productRoutes.GET("", productHandler.GetAll)
		productRoutes.GET("/:id", productHandler.GetByID)
	}

	log.Println("Product service running on port :8082")
	if err := r.Run(":8082"); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
