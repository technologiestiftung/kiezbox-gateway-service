package routes

import (
	"kiezbox/api/handlers"

	"github.com/gin-gonic/gin"
)

func CORSMiddleware() gin.HandlerFunc {
    //TODO: Cors middleware only used to make local tests work. make this toggled by cli/build flag
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

func RegisterRoutes(r *gin.Engine) {
	// Register the hello world route
	r.Use(CORSMiddleware())
	r.GET("/mode", handlers.Mode)
	r.Any("/session", handlers.Session)
	r.GET("/sipconfig", handlers.SIPConfig)
	r.POST("/asterisk/:pstype/:singlemulti", handlers.Asterisk)
}
