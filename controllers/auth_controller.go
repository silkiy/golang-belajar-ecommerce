package controllers

import (
	"context"
	"ecommerce/database"
	"ecommerce/models"
	"net/http"
	"os"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret = []byte(os.Getenv("JWT_SECRET"))

func Register(c *gin.Context) {
	var input struct {
		Name     string `json:"name" binding:"required"`
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
		Role     string `json:"role"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var existingUser models.User
	err := database.UserCollection.FindOne(ctx, bson.M{"email": input.Email}).Decode(&existingUser)
	if err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Email already registered"})
		return
	}

	hashed, _ := bcrypt.GenerateFromPassword([]byte(input.Password), 10)

	role := input.Role
	if role == "" {
		role = "customer"
	}

	user := models.User{
		ID:       primitive.NewObjectID(),
		Name:     input.Name,
		Email:    input.Email,
		Password: string(hashed),
		Role:     role,
		CreatedAt: time.Now(),
	}

	_, err = database.UserCollection.InsertOne(ctx, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User registered successfully",
		"user": gin.H{
			"id":    user.ID.Hex(),
			"name":  user.Name,
			"email": user.Email,
			"role":  user.Role,
		},
	})
}

func Login(c *gin.Context) {
	var input struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	err := database.UserCollection.FindOne(ctx, bson.M{"email": input.Email}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password))
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid email or password"})
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"userId": user.ID.Hex(),
		"role":   user.Role,
		"exp":    time.Now().Add(time.Hour * 24).Unix(),
	})

	tokenString, _ := token.SignedString(jwtSecret)

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":    user.ID.Hex(),
			"name":  user.Name,
			"email": user.Email,
			"role":  user.Role,
			"token": tokenString,
		},
	})
}

func Logout(c *gin.Context) {
    tokenString := c.GetHeader("Authorization")
    if tokenString == "" {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Token required"})
        return
    }

    if len(tokenString) > 7 && tokenString[:7] == "Bearer " {
        tokenString = tokenString[7:]
    }

    token, _ := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
        return jwtSecret, nil
    })

    if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
        exp := int64(claims["exp"].(float64))

        ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()

        _, err := database.DB.Collection("blacklist_tokens").InsertOne(ctx, bson.M{
            "token": tokenString,
            "exp":   exp,
        })
        if err != nil {
            c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to blacklist token"})
            return
        }

        c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully"})
        return
    }

    c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
}
