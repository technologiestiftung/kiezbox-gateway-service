package routes

import (
	"kiezbox/api/handlers"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {
	// Register the hello world route
	r.GET("/hello", handlers.HelloWorld)
	r.GET("/mode", handlers.Mode)
	r.POST("/validateSession", handlers.ValidateSession)
	r.GET("/session", handlers.Session)
	r.POST("/asterisk/:pstype/:singlemulti", handlers.Asterisk)
}
