package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID        primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	Name      string              `bson:"name" json:"name"`
	Email     string              `bson:"email" json:"email"`
	Password  string              `bson:"password" json:"-"`
	Role      string              `bson:"role" json:"role"`
	CreatedAt time.Time `bson:"createdAt" json:"createdAt"`
}
