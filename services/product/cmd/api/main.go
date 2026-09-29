package main

import (
	"ecommerce/services/product/internal/config"
	"ecommerce/services/product/internal/handler"
	"ecommerce/services/product/internal/repository"
	"ecommerce/services/product/internal/service"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found")
	}

	db := config.ConnectDB()

	productRepo := repository.NewProductRepository(db)
	productService := service.NewProductService(productRepo)
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
