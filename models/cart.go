package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type CartItem struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    primitive.ObjectID `bson:"userId" json:"userId"`
	ProductID primitive.ObjectID `bson:"productId" json:"productId"`
	Quantity  int                `bson:"quantity" json:"quantity"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
}
