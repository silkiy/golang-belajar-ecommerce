package controllers

import (
	"context"
	"ecommerce/database"
	"ecommerce/models"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func GetOrdersAdmin(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := database.OrderCollection.Find(ctx, bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var orders []models.Order
	cursor.All(ctx, &orders)

	c.JSON(http.StatusOK, gin.H{"message": "Fetch success", "data": orders})
}

func GetOrderByIDAdmin(c *gin.Context) {
	orderId := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(orderId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var order models.Order
	err = database.OrderCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&order)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Fetch success", "data": order})
}

func UpdateOrderStatus(c *gin.Context) {
	orderId := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(orderId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	var body struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
		return
	}

	allowedStatuses := []string{"pending", "paid", "delivered", "canceled", "refunded", "completed"}
	isValid := false
	for _, s := range allowedStatuses {
		if body.Status == s {
			isValid = true
			break
		}
	}
	if !isValid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid status value"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var existingOrder models.Order
	err = database.OrderCollection.FindOne(ctx, bson.M{"_id": objID}).Decode(&existingOrder)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Order not found"})
		return
	}

	validTransitions := map[string][]string{
		"pending":   {"paid", "canceled"},
		"paid":      {"delivered", "refunded"},
		"delivered": {"completed"},
		"canceled":  {},
		"refunded":  {},
		"completed": {},
	}

	currentStatus := existingOrder.Status
	allowedNext := validTransitions[currentStatus]

	canUpdate := false
	for _, s := range allowedNext {
		if body.Status == s {
			canUpdate = true
			break
		}
	}
	if !canUpdate {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Cannot change status from %s to %s", currentStatus, body.Status),
		})
		return
	}

	update := bson.M{
		"$set": bson.M{
			"status":    body.Status,
			"updatedAt": time.Now(),
		},
	}
	opts := options.FindOneAndUpdate().SetReturnDocument(options.After)

	var updatedOrder models.Order
	err = database.OrderCollection.FindOneAndUpdate(ctx, bson.M{"_id": objID}, update, opts).Decode(&updatedOrder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update order"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Order status updated",
		"data":    updatedOrder,
	})
}

func CancelOrderAdmin(c *gin.Context) {
	orderId := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(orderId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid order ID"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"_id": objID, "status": bson.M{"$in": []string{"pending", "paid"}}}
	update := bson.M{"$set": bson.M{"status": "canceled", "updatedAt": time.Now()}}

	result, err := database.OrderCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel order"})
		return
	}
	if result.MatchedCount == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order cannot be canceled"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Order canceled"})
}
