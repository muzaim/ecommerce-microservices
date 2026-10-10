package handler

import (
	"net/http"
	"strconv"

	"ecommerce/services/auth/internal/model"
	"ecommerce/services/auth/internal/service"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	authService service.AuthService
}

func NewAuthHandler(authService service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req model.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.authService.Register(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if res.SessionID != "" {
		c.SetCookie("session_id", res.SessionID, 86400, "/", "", false, true)
	}

	c.JSON(http.StatusCreated, res)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	res, err := h.authService.Login(req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	if res.SessionID != "" {
		c.SetCookie("session_id", res.SessionID, 86400, "/", "", false, true)
	}

	c.JSON(http.StatusOK, res)
}

func (h *AuthHandler) GetProfile(c *gin.Context) {
	var userID uint

	// 1. Baca X-User-ID yang di-inject oleh API Gateway dari Redis Session / JWT Check
	userIDHeader := c.GetHeader("X-User-ID")
	if userIDHeader != "" {
		if uID, err := strconv.ParseUint(userIDHeader, 10, 32); err == nil {
			userID = uint(uID)
		}
	}

	// 2. Fallback jika ada context userID lokal
	if userID == 0 {
		if val, exists := c.Get("userID"); exists {
			if uID, ok := val.(uint); ok {
				userID = uID
			}
		}
	}

	if userID == 0 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized: missing user session"})
		return
	}

	user, err := h.authService.GetProfile(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, user)
}
