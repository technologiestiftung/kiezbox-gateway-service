package handlers

import (
	cfg "kiezbox/internal/config"
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
func Mode(ctx *gin.Context) {
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
