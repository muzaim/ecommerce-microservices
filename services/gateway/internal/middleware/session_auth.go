package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

type SessionData struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
}

func SessionOrJWTAuthMiddleware(rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		var sessionID string

		// 1. Cek dari Cookie "session_id" (Sangat aman untuk Web)
		cookieSessionID, err := c.Cookie("session_id")
		if err == nil && cookieSessionID != "" {
			sessionID = cookieSessionID
		}

		// 2. Cek dari Header "X-Session-ID" atau "Authorization: Bearer <token/session_id>"
		if sessionID == "" {
			sessionID = c.GetHeader("X-Session-ID")
		}

		authHeader := c.GetHeader("Authorization")
		var tokenStr string
		if authHeader != "" {
			parts := strings.Split(authHeader, " ")
			if len(parts) == 2 && parts[0] == "Bearer" {
				tokenStr = parts[1]
				if sessionID == "" {
					sessionID = tokenStr
				}
			}
		}

		// 3. Cek ke Redis Store terlebih dahulu (Stateful Session Check)
		if rdb != nil && sessionID != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			val, err := rdb.Get(ctx, fmt.Sprintf("session:%s", sessionID)).Result()
			if err == nil && val != "" {
				var data SessionData
				if err := json.Unmarshal([]byte(val), &data); err == nil && data.UserID != 0 {
					userIDStr := fmt.Sprintf("%d", data.UserID)
					c.Request.Header.Set("X-User-ID", userIDStr)
					c.Set("userID", userIDStr)
					c.Next()
					return
				}
			}
		}

		// 4. Fallback ke Stateless JWT Validation jika Redis session tidak ditemukan
		if tokenStr != "" {
			secretKey := []byte(os.Getenv("JWT_SECRET"))
			token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return secretKey, nil
			})

			if err == nil && token.Valid {
				if claims, ok := token.Claims.(jwt.MapClaims); ok {
					if userIDFloat, ok := claims["user_id"].(float64); ok {
						userIDStr := fmt.Sprintf("%d", uint(userIDFloat))
						c.Request.Header.Set("X-User-ID", userIDStr)
						c.Set("userID", userIDStr)
						c.Next()
						return
					}
				}
			}
		}

		// 5. Jika tidak ada Session ID di Redis dan tidak ada JWT valid -> Reject 401!
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Unauthorized: Session expired or login required",
		})
		c.Abort()
	}
}
