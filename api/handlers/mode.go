package handlers

import (
	"math/rand"
	"net/http"

	"github.com/gin-gonic/gin"
)

// @Summary Get current mode
// @Description Returns the current Kiezbox mode
// @Tags Mode
// @Produce json
// @Success 200 {object} map[string]string
// @Router /mode [get]
func Mode(c *gin.Context) {
	// random number for demonstration purposes 1-3
	mode := rand.Intn(3)
	c.JSON(http.StatusOK, gin.H{
		"mode": mode, // e.g., "normal", "maintenance"
	})
}
