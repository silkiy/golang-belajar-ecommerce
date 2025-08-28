package main

import (
	"ecommerce/config"
	"ecommerce/database"
	"ecommerce/routes"

	"github.com/gin-gonic/gin"
)

func main() {

	config.LoadEnv()

	database.ConnectMongo()
	database.InitCollections()

	r := gin.Default()
	r.SetTrustedProxies(nil)
	routes.RegisterRoutes(r)

	port := config.GetEnv("PORT", "8080")
	r.Run(":" + port)
}
