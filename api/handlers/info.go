package handlers

import (
	cfg "kiezbox/internal/config"
	"net/http"

	"github.com/gin-gonic/gin"
)

// @Summary Get general Info, usually only once per user
// Currently returns box location (lon,lat)
func Info(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, gin.H{
		"lon": cfg.Cfg.BoxLon,
		"lat": cfg.Cfg.BoxLat,
	})
}
