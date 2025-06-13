package handlers

import (
	"context"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"sync"

	"kiezbox/internal/meshtastic"

	"github.com/gin-gonic/gin"
)

// SetModeRequest represents the expected JSON body
type SetModeRequest struct {
	Mode int `json:"mode"`
}

// GetMode fetches the current mode
func GetMode(c *gin.Context) {
	// random number for demonstration purposes 1-3
	var mode int
	value := os.Getenv("KB_MODE_OVERRIDE")
	if value == "" {
		mode = rand.Intn(3)
	} else {
		converted, err := strconv.Atoi(value)
		if err == nil {
			mode = converted
		} else {
			mode = rand.Intn(3)
		}
	}
	c.JSON(http.StatusOK, gin.H{
		"mode": mode, // e.g., "normal", "maintenance"
	})
}

// SetMode sets the Kiezbox mode to the provided value
func SetMode(mts *meshtastic.MTSerial, ctx context.Context, wg *sync.WaitGroup) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Extract the "mode" parameter from the URL
		modeStr := c.Param("mode")
		modeInt, err := strconv.Atoi(modeStr)
		if err != nil || modeInt < 0 || modeInt > 2 {
			// Respond with error if the mode is not a valid integer
			c.JSON(400, gin.H{"error": "Invalid mode. Allowed values: 0, 1, 2"})
			return
		}
		mode := int32(modeInt)

		// Build the control message to set the desired mode
		control := meshtastic.BuildKiezboxControl(nil, &mode)
		// Set the mode
		go mts.SetKiezboxValues(ctx, wg, control)
		// Reply to the client with success
		c.JSON(http.StatusOK, gin.H{"status": "mode set", "mode": mode})
	}
}
