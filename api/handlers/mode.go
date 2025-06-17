package handlers

import (
	"math/rand"
	"net/http"

	cfg "kiezbox/internal/config"

	"github.com/gin-gonic/gin"
)

// GetMode fetches the current mode
func GetMode(ctx *gin.Context) {
	// random number for demonstration purposes 1-3
	var mode int
	if cfg.Cfg.ModeOverride {
		mode = cfg.Cfg.Mode
	} else {
		//TODO: implement retrieving real mode
		// random number for demonstration purposes 1-3
		mode = rand.Intn(3)
	}
	ctx.JSON(http.StatusOK, gin.H{
		"mode": mode, // e.g., "normal", "maintenance"
	})
}
