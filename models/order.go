package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Order struct {
	ID        primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	UserID    primitive.ObjectID   `bson:"userId" json:"userId"`
	Products  []OrderItem          `bson:"products" json:"products"`
	Total     float64              `bson:"total" json:"total"`
	Status    string               `bson:"status" json:"status"`
	CreatedAt int64                `bson:"createdAt" json:"createdAt"`
}

type OrderItem struct {
	ProductID primitive.ObjectID `bson:"productId" json:"productId"`
	Quantity  int                `bson:"quantity" json:"quantity"`
	Price     float64            `bson:"price" json:"price"`
}
