package main

import (
	"log"
	"net/http"

	"ecommerce/services/auth/internal/config"
	"ecommerce/services/auth/internal/handler"
	"ecommerce/services/auth/internal/middleware"
	"ecommerce/services/auth/internal/repository"
	"ecommerce/services/auth/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	db := config.ConnectDB()

	// Dependency Injection: DB -> Repo -> Service -> Handler
	userRepo := repository.NewUserRepository(db)
	authService := service.NewAuthService(userRepo)
	authHandler := handler.NewAuthHandler(authService)

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "OK",
			"service": "auth-service",
		})
	})

	// Routes Auth
	authRoutes := r.Group("/auth")
	{
		authRoutes.POST("/register", authHandler.Register)
		authRoutes.POST("/login", authHandler.Login)
	}

	// Routes Auth Protected (Butuh JWT Token)
	protected := r.Group("/auth")
	protected.Use(middleware.JWTAuthMiddleware())
	{
		protected.GET("/me", authHandler.GetProfile)
	}

	log.Println("Auth service running on port :8081")
	if err := r.Run(":8081"); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
