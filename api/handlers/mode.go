package handlers

import (
	"net/http"

	cfg "kiezbox/internal/config"
	"kiezbox/internal/state"

	"github.com/gin-gonic/gin"
)

// GetMode fetches the current mode
func GetMode(ctx *gin.Context) {
	var mode int
	if cfg.Cfg.ModeOverride {
		mode = cfg.Cfg.Mode
	} else {
		mode = state.GetMode()
	}
	ctx.JSON(http.StatusOK, gin.H{
		"mode": mode,
	})
}
