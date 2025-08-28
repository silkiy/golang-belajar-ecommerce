package database

import (
	"context"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var Client *mongo.Client
var DB *mongo.Database

func ConnectMongo() {
	uri := os.Getenv("MONGO_URI")
	dbName := os.Getenv("DB_NAME")

	if uri == "" || dbName == "" {
		log.Fatal("❌ MONGO_URI or DB_NAME not set in environment variables")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		log.Fatal("❌ MongoDB connection error:", err)
	}

	Client = client
	DB = client.Database(dbName)

	log.Println("✅ Connected to MongoDB Atlas")
}

var UserCollection *mongo.Collection
var ProductCollection *mongo.Collection
var OrderCollection *mongo.Collection
var CartCollection *mongo.Collection

func InitCollections() {
	UserCollection = DB.Collection("users")
	ProductCollection = DB.Collection("products")
	OrderCollection = DB.Collection("orders")
	CartCollection = DB.Collection("carts")
}
