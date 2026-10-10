package handler

import (
	"net/http"
	"strconv"

	"ecommerce/services/order/internal/model"
	"ecommerce/services/order/internal/service"

	"github.com/gin-gonic/gin"
)

type OrderHandler struct {
	service service.OrderService
}

func NewOrderHandler(service service.OrderService) *OrderHandler {
	return &OrderHandler{service: service}
}

func (h *OrderHandler) Create(c *gin.Context) {
	var req model.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userIDHeader := c.GetHeader("X-User-ID")
	if userIDHeader != "" {
		if uID, err := strconv.ParseUint(userIDHeader, 10, 32); err == nil {
			req.UserID = uint(uID)
		}
	}

	if req.UserID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User ID is required"})
		return
	}

	res, err := h.service.CreateOrder(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, res)
}

func (h *OrderHandler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	order, err := h.service.GetOrderByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	c.JSON(http.StatusOK, order)
}

func (h *OrderHandler) GetMyOrders(c *gin.Context) {
	var userID uint

	userIDHeader := c.GetHeader("X-User-ID")
	if userIDHeader != "" {
		if uID, err := strconv.ParseUint(userIDHeader, 10, 32); err == nil {
			userID = uint(uID)
		}
	}

	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User ID is required"})
		return
	}

	orders, err := h.service.GetMyOrders(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, orders)
}
