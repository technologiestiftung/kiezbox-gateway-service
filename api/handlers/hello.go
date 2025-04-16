package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// @Summary Say hello
// @Description Returns a simple hello world message
// @Tags test
// @Produce json
// @Success 200 {string} string "Hello, World!"
// @Router /hello [get]
func HelloWorld(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"message": "Hello, World!",
	})
}
