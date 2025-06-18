package handlers

import (
	"net/http"

	"kiezbox/internal/state"

	"github.com/gin-gonic/gin"
)

// GetMode fetches the current mode
func GetMode(ctx *gin.Context) {
	mode := state.GetMode()
	ctx.JSON(http.StatusOK, gin.H{
		"mode": mode,
	})
}
