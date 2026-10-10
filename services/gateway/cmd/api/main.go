package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"ecommerce/services/gateway/internal/config"
	"ecommerce/services/gateway/internal/middleware"
	"ecommerce/services/gateway/internal/proxy"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	authServiceURL := os.Getenv("AUTH_SERVICE_URL")
	productServiceURL := os.Getenv("PRODUCT_SERVICE_URL")
	orderServiceURL := os.Getenv("ORDER_SERVICE_URL")

	rdb := config.ConnectRedis()

	r := gin.Default()

	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "OK",
			"service": "api-gateway",
		})
	})

	authProxy := proxy.ReverseProxyHandler(authServiceURL)
	productProxy := proxy.ReverseProxyHandler(productServiceURL)
	orderProxy := proxy.ReverseProxyHandler(orderServiceURL)

	// =========================================================================
	// 1. PUBLIC ROUTES (HANYA Register & Login)
	// =========================================================================
	publicAuth := r.Group("/api/auth")
	{
		publicAuth.POST("/register", func(c *gin.Context) {
			c.Request.URL.Path = "/auth/register"
			authProxy(c)
		})
		publicAuth.POST("/login", func(c *gin.Context) {
			c.Request.URL.Path = "/auth/login"
			authProxy(c)
		})
	}

	// =========================================================================
	// 2. PROTECTED ROUTES (WAJIB REDIS SESSION ATAU JWT TOKEN)
	// =========================================================================
	protected := r.Group("/api")
	protected.Use(middleware.SessionOrJWTAuthMiddleware(rdb))
	{
		// Logout Route (Hapus Session dari Redis & Clear Cookie)
		protected.POST("/auth/logout", func(c *gin.Context) {
			sessionID, _ := c.Cookie("session_id")
			if sessionID == "" {
				sessionID = c.GetHeader("X-Session-ID")
			}
			if sessionID == "" {
				authHeader := c.GetHeader("Authorization")
				parts := strings.Split(authHeader, " ")
				if len(parts) == 2 {
					sessionID = parts[1]
				}
			}

			if rdb != nil && sessionID != "" {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				rdb.Del(ctx, fmt.Sprintf("session:%s", sessionID))
			}

			// Clear Cookie
			c.SetCookie("session_id", "", -1, "/", "", false, true)
			c.JSON(http.StatusOK, gin.H{
				"message": "Successfully logged out. Session deleted from Redis.",
			})
		})

		// Auth Protected Routes (/api/auth/me)
		protected.GET("/auth/me", func(c *gin.Context) {
			c.Request.URL.Path = "/auth/me"
			authProxy(c)
		})

		// Product Protected Routes (/api/products & /api/products/*path)
		handleProductProxy := func(c *gin.Context) {
			c.Request.URL.Path = "/products" + c.Param("path")
			productProxy(c)
		}
		protected.Any("/products", handleProductProxy)
		protected.Any("/products/*path", handleProductProxy)

		// Order Protected Routes (/api/orders & /api/orders/*path)
		handleOrderProxy := func(c *gin.Context) {
			c.Request.URL.Path = "/orders" + c.Param("path")
			orderProxy(c)
		}
		protected.Any("/orders", handleOrderProxy)
		protected.Any("/orders/*path", handleOrderProxy)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	log.Printf("API Gateway running on port :%s\n", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to run API Gateway: %v", err)
	}
}
