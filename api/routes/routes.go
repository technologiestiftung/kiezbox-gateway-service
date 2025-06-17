package routes

import (
	"context"
	"kiezbox/api/handlers"
	"kiezbox/internal/meshtastic"
	"sync"

	cfg "kiezbox/internal/config"

	"github.com/gin-gonic/gin"
)

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, HEAD, PUT")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func RegisterRoutes(r *gin.Engine, mts *meshtastic.MTSerial, ctx context.Context, wg *sync.WaitGroup) {
	// Use Corse middlewar only for local testing
	if cfg.Cfg.CorsLocalhost {
		r.Use(CORSMiddleware())
	}
	r.GET("/mode", handlers.GetMode)
	r.GET("/info", handlers.Info)
	r.Any("/session", handlers.Session)
	r.POST("/asterisk/:pstype/:singlemulti", handlers.Asterisk)
	r.POST("/admin/control", handlers.SetKiezboxControlValue(mts, ctx, wg))
}
