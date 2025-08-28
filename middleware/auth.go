package middleware

import (
	"context"
	"ecommerce/database"
	"net/http"
	"os"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        tokenString := c.GetHeader("Authorization")
        if tokenString == "" {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token required"})
            return
        }

        if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
            tokenString = tokenString[7:]
        }

        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()

        var blacklisted bson.M
        err := database.DB.Collection("blacklist_tokens").FindOne(ctx, bson.M{"token": tokenString}).Decode(&blacklisted)
        if err == nil {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Token has been blacklisted"})
            return
        }

        token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
            return jwtSecret, nil
        })

        if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
            c.Set("userId", claims["userId"])
            c.Set("role", claims["role"])
            c.Next()
        } else {
            c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
        }
    }
}

func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists || role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: admin only"})
			c.Abort()
			return
		}
		c.Next()
	}
}
