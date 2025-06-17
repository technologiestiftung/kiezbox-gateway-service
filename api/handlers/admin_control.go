package handlers

import (
	"context"
	"net/http"
	"sync"

	"kiezbox/internal/meshtastic"

	"github.com/gin-gonic/gin"
)

// SetKiezboxControlValue sets a Kiezbox control value based on the provided key and value
func SetKiezboxControlValue(mts *meshtastic.MTSerial, ctx context.Context, wg *sync.WaitGroup) gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		// Extract key and value parameters from the query string
		key := ginCtx.Query("key")
		value := ginCtx.Query("value")
		if key == "" || value == "" {
			ginCtx.JSON(400, gin.H{"error": "Missing key or value query parameter."})
			return
		}

		// Build the control message for the provided key and value
		control := meshtastic.BuildKiezboxControlMessage(key, value)
		if control == nil {
			ginCtx.JSON(400, gin.H{"error": "Invalid key or value."})
			return
		}

		// Set the control value
		go mts.SetKiezboxControlValue(ctx, wg, control)
		// Reply to the client with success
		ginCtx.JSON(http.StatusOK, gin.H{"status": "control value set", "key": key, "value": value})
	}
}
