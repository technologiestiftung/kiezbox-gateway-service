package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// HelloWorld Returns a simple hello world message
func HelloWorld(c *gin.Context) {
	c.String(http.StatusOK, "Hello, World!")
}
