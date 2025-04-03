package main

import (
	_ "kiezbox/api/docs" // Import the generated Swagger docs
	"kiezbox/api/routes"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// @title Kiezbox API
// @version 1.0
// @description API for managing communication in Kiezbox crisis mode.

func main() {
	r := gin.Default()

	// Register API routes
	routes.RegisterRoutes(r)

	// Serve Swagger UI at /swagger
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Start the server
	r.Run(":8080")
}
