package controllers

import (
	"context"
	"ecommerce/database"
	"ecommerce/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func AddToCart(c *gin.Context) {
    var body struct {
        ProductID string `json:"productId"`
        Quantity  int    `json:"quantity"`
    }
    if err := c.ShouldBindJSON(&body); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
        return
    }

    userId, _ := c.Get("userId")
    objUserID, _ := primitive.ObjectIDFromHex(userId.(string))
    objProductID, _ := primitive.ObjectIDFromHex(body.ProductID)

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    var product models.Product
    err := database.ProductCollection.FindOne(ctx, bson.M{"_id": objProductID}).Decode(&product)
    if err != nil {
        c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
        return
    }

    if body.Quantity > product.Stock {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Quantity exceeds available stock"})
        return
    }

    cartItem := models.CartItem{
        ID:        primitive.NewObjectID(),
        UserID:    objUserID,
        ProductID: objProductID,
        Quantity:  body.Quantity,
        CreatedAt: time.Now(),
    }

    _, err = database.CartCollection.InsertOne(ctx, cartItem)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add to cart"})
        return
    }

    response := gin.H{
        "cartId":    cartItem.ID,
        "productId": cartItem.ProductID,
        "quantity":  cartItem.Quantity,
        "createdAt": cartItem.CreatedAt,
        "product": gin.H{
            "name":  product.Name,
            "price": product.Price,
            "stock": product.Stock,
        },
        "subtotal": float64(cartItem.Quantity) * product.Price,
    }

    c.JSON(http.StatusOK, gin.H{"message": "Added to cart", "data": response})
}

func GetCart(c *gin.Context) {
    userId, _ := c.Get("userId")
    objUserID, _ := primitive.ObjectIDFromHex(userId.(string))

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    cursor, err := database.CartCollection.Find(ctx, bson.M{"userId": objUserID})
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    var cartItems []models.CartItem
    if err := cursor.All(ctx, &cartItems); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }

    var cartWithProducts []gin.H
    for _, item := range cartItems {
        var product models.Product
        err := database.ProductCollection.FindOne(ctx, bson.M{"_id": item.ProductID}).Decode(&product)
        if err != nil {
            continue
        }

        cartWithProducts = append(cartWithProducts, gin.H{
            "productId":   item.ProductID,
            "quantity":    item.Quantity,
            "productName": product.Name,
            "price":       product.Price,
            "total":       float64(item.Quantity) * product.Price,
        })
    }

    c.JSON(http.StatusOK, gin.H{"message": "Fetch success", "data": cartWithProducts})
}

func UpdateCart(c *gin.Context) {
	userIDHex := c.MustGet("userId").(string)
	userID, _ := primitive.ObjectIDFromHex(userIDHex)

	productId := c.Param("productId")
	productObjID, err := primitive.ObjectIDFromHex(productId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid productId"})
		return
	}

	var body struct {
		Quantity int `json:"quantity"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Quantity < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid quantity"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var cartItem models.CartItem
	err = database.CartCollection.FindOne(ctx, bson.M{"userId": userID, "productId": productObjID}).Decode(&cartItem)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusNotFound, gin.H{"error": "Product not found in cart"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch cart"})
		}
		return
	}

	// Ambil data product
	var product models.Product
	if err := database.ProductCollection.FindOne(ctx, bson.M{"_id": productObjID}).Decode(&product); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found"})
		return
	}

	if body.Quantity == 0 {
		_, err := database.CartCollection.DeleteOne(ctx, bson.M{"userId": userID, "productId": productObjID})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to remove product from cart"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "Product removed from cart"})
		return
	}

	if body.Quantity > product.Stock {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Quantity exceeds available stock"})
		return
	}

	filter := bson.M{"userId": userID, "productId": productObjID}
	update := bson.M{"$set": bson.M{"quantity": body.Quantity}}

	_, err = database.CartCollection.UpdateOne(ctx, filter, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update cart"})
		return
	}

	response := gin.H{
		"productId": productObjID,
		"quantity":  body.Quantity,
		"product": gin.H{
			"name":  product.Name,
			"price": product.Price,
			"stock": product.Stock,
		},
		"subtotal": float64(body.Quantity) * product.Price,
	}

	c.JSON(http.StatusOK, gin.H{"message": "Cart updated", "data": response})
}

func RemoveFromCart(c *gin.Context) {
	userIDHex := c.MustGet("userId").(string)
	userID, _ := primitive.ObjectIDFromHex(userIDHex)

	productId := c.Param("productId")
	productObjID, err := primitive.ObjectIDFromHex(productId)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid productId"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := database.CartCollection.DeleteOne(ctx, bson.M{
		"userId":    userID,
		"productId": productObjID,
	})
	if err != nil || result.DeletedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Product not found in cart"})
		return
	}

	var product models.Product
	if err := database.ProductCollection.FindOne(ctx, bson.M{"_id": productObjID}).Decode(&product); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"message":   "Product removed from cart",
			"productId": productObjID.Hex(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Product removed from cart",
		"data": gin.H{
			"productId": productObjID,
			"name":      product.Name,
			"price":     product.Price,
		},
	})
}
