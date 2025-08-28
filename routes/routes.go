package routes

import (
	"ecommerce/controllers"
	"ecommerce/middleware"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {

	api := r.Group("/api")
	{
		api.POST("/register", controllers.Register)
		api.POST("/login", controllers.Login)
		api.POST("/logout", controllers.Logout)

		protected := api.Group("/")
		protected.Use(middleware.AuthMiddleware())
		{
			admin := protected.Group("/admin")
			admin.Use(middleware.AdminMiddleware())
			{
				admin.POST("/products", controllers.CreateProduct)
				admin.PUT("/products/:id", controllers.UpdateProduct)
				admin.DELETE("/products/:id", controllers.DeleteProduct)
				admin.GET("/products", controllers.GetProductsAdmin)

				admin.GET("/orders", controllers.GetOrdersAdmin)
				admin.GET("/orders/:id", controllers.GetOrderByIDAdmin)
				admin.PUT("/orders/:id/status", controllers.UpdateOrderStatus)
				admin.PUT("/orders/:id/cancel", controllers.CancelOrderAdmin)
			}

			user := protected.Group("/user")
			{
				user.GET("/products", controllers.GetProductsPublic)

				user.POST("/cart", controllers.AddToCart)
				user.GET("/cart", controllers.GetCart)
				user.PUT("/cart/:productId", controllers.UpdateCart)
				user.DELETE("/cart/:productId", controllers.RemoveFromCart)

				user.POST("/checkout", controllers.Checkout)
				user.GET("/orders", controllers.GetOrders)
				user.PUT("/orders/:id/cancel", controllers.CancelOrder)
			}
		}
	}
}
