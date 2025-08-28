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
)

func Checkout(c *gin.Context) {
	userId, _ := c.Get("userId")
	objUserID, _ := primitive.ObjectIDFromHex(userId.(string))

	var body struct {
		ProductIDs []string `json:"productIds"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || len(body.ProductIDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid productIds"})
		return
	}

	var objIDs []primitive.ObjectID
	for _, pid := range body.ProductIDs {
		oid, err := primitive.ObjectIDFromHex(pid)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid productId format"})
			return
		}
		objIDs = append(objIDs, oid)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := database.CartCollection.Find(ctx, bson.M{
		"userId":    objUserID,
		"productId": bson.M{"$in": objIDs},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch cart"})
		return
	}

	var cartItems []models.CartItem
	if err := cursor.All(ctx, &cartItems); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode cart"})
		return
	}

	if len(cartItems) != len(objIDs) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "One or more products are not in your cart",
		})
		return
	}

	type ProductDetail struct {
		ID       primitive.ObjectID `json:"id"`
		Name     string             `json:"name"`
		Price    float64            `json:"price"`
		Quantity int                `json:"quantity"`
	}

	var orderItems []models.OrderItem
	var productDetails []ProductDetail
	var total float64
	var updatedProducts []struct {
		ProductID primitive.ObjectID
		Quantity  int
	}

	for _, item := range cartItems {
		var product models.Product
		err := database.ProductCollection.FindOne(ctx, bson.M{"_id": item.ProductID}).Decode(&product)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Product not found"})
			return
		}
		if item.Quantity > product.Stock {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Not enough stock for %s, available: %d", product.Name, product.Stock),
			})
			return
		}
	}

	for _, item := range cartItems {
		var product models.Product
		_ = database.ProductCollection.FindOne(ctx, bson.M{"_id": item.ProductID}).Decode(&product)

		_, err = database.ProductCollection.UpdateOne(
			ctx,
			bson.M{"_id": product.ID, "stock": bson.M{"$gte": item.Quantity}},
			bson.M{"$inc": bson.M{"stock": -item.Quantity}},
		)
		if err != nil {
			rollbackStock(ctx, updatedProducts)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update stock"})
			return
		}

		updatedProducts = append(updatedProducts, struct {
			ProductID primitive.ObjectID
			Quantity  int
		}{ProductID: product.ID, Quantity: item.Quantity})

		orderItems = append(orderItems, models.OrderItem{
			ProductID: item.ProductID,
			Quantity:  item.Quantity,
			Price:     product.Price,
		})

		productDetails = append(productDetails, ProductDetail{
			ID:       product.ID,
			Name:     product.Name,
			Price:    product.Price,
			Quantity: item.Quantity,
		})

		total += product.Price * float64(item.Quantity)
	}

	order := models.Order{
		ID:        primitive.NewObjectID(),
		UserID:    objUserID,
		Products:  orderItems,
		Total:     total,
		Status:    "pending",
		CreatedAt: time.Now().Unix(),
	}

	_, err = database.OrderCollection.InsertOne(ctx, order)
	if err != nil {
		rollbackStock(ctx, updatedProducts)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create order"})
		return
	}

	_, _ = database.CartCollection.DeleteMany(ctx, bson.M{
		"userId":    objUserID,
		"productId": bson.M{"$in": objIDs},
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Checkout success",
		"order": gin.H{
			"id":        order.ID.Hex(),
			"userId":    order.UserID.Hex(),
			"total":     order.Total,
			"status":    order.Status,
			"products":  productDetails,
			"createdAt": order.CreatedAt,
		},
	})
}

func GetOrders(c *gin.Context) {
	userId, _ := c.Get("userId")
	objUserID, _ := primitive.ObjectIDFromHex(userId.(string))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := database.OrderCollection.Find(ctx, bson.M{"userId": objUserID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var orders []models.Order
	if err := cursor.All(ctx, &orders); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	type ProductDetail struct {
		ID       primitive.ObjectID `json:"id"`
		Name     string             `json:"name"`
		Price    float64            `json:"price"`
		Quantity int                `json:"quantity"`
	}

	var resp []gin.H
	for _, order := range orders {
		var products []ProductDetail
		for _, item := range order.Products {
			var product models.Product
			err := database.ProductCollection.FindOne(ctx, bson.M{"_id": item.ProductID}).Decode(&product)
			if err != nil {
				continue
			}

			products = append(products, ProductDetail{
				ID:       product.ID,
				Name:     product.Name,
				Price:    product.Price,
				Quantity: item.Quantity,
			})
		}

		resp = append(resp, gin.H{
			"id":        order.ID.Hex(),
			"total":     order.Total,
			"status":    order.Status,
			"products":  products,
			"createdAt": order.CreatedAt,
		})
	}

	c.JSON(http.StatusOK, gin.H{"message": "Fetch success", "data": resp})
}

func CancelOrder(c *gin.Context) {
	userId, _ := c.Get("userId")
	objUserID, err := primitive.ObjectIDFromHex(userId.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid userId"})
		return
	}

	orderId := c.Param("id")
	orderObjID, err := primitive.ObjectIDFromHex(orderId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid orderId"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{
		"_id":    orderObjID,
		"userId": objUserID,
		"status": "pending",
	}
	update := bson.M{
		"$set": bson.M{"status": "canceled"},
	}

	result, err := database.OrderCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel order"})
		return
	}
	if result.MatchedCount == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Order not found or cannot be canceled"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Order canceled"})
}

func rollbackStock(ctx context.Context, updated []struct {
	ProductID primitive.ObjectID
	Quantity  int
}) {
	for _, u := range updated {
		_, _ = database.ProductCollection.UpdateOne(
			ctx,
			bson.M{"_id": u.ProductID},
			bson.M{"$inc": bson.M{"stock": u.Quantity}},
		)
	}
}