package routes

import (
	"kiezbox/api/handlers"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {
	// Register the hello world route
	r.GET("/hello", handlers.HelloWorld)
}
