package handlers

import (
	"context"
	"net/http"
	"sync"

	"kiezbox/internal/meshtastic"

	"github.com/gin-gonic/gin"
)

// SetKiezboxControlValue sets a Kiezbox control value based on the provided key and value
func SetKiezboxControlValue(device meshtastic.MeshtasticDevice, ctx context.Context, wg *sync.WaitGroup) gin.HandlerFunc {
	return func(ginCtx *gin.Context) {
		// Extract key and value parameters from the query string
		key := ginCtx.PostForm("key")
		value := ginCtx.PostForm("value")
		filter := []string{ginCtx.PostForm("box_id"),ginCtx.PostForm("dist_id"),ginCtx.PostForm("sens_id"),ginCtx.PostForm("dev_type")}
		if key == "" || value == "" {
			ginCtx.JSON(400, gin.H{"error": "Missing key or value query parameter."})
			return
		}

		// Build the control message for the provided key and value
		control := meshtastic.BuildKiezboxControlMessage(key, value, filter)
		if control == nil {
			ginCtx.JSON(400, gin.H{"error": "Invalid key or value."})
			return
		}

		// Set the control value
		go device.SetKiezboxControlValue(ctx, wg, control)
		// Reply to the client with success
		ginCtx.JSON(http.StatusOK, gin.H{"status": "control value set", "key": key, "value": value})
	}
}
