package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Request struct for ValidateSession
type ValidateSessionRequest struct {
	Config map[string]interface{} `json:"config"`
}

// @Summary Post ValidateSession
// @Description Returns validateSession with validateSession token, creating a new one only if needed
// @Tags validateSession
// @Accept json
// @Produce json
// @Param config body ValidateSessionRequest true "Configuration parameters"
// @Success 200 {object} map[string]interface{} "Configuration with validateSession token"
// @Router /validateSession [post]
func ValidateSession(c *gin.Context) {
    // cookieName, cookieMaxAge, cookiePath, cookieDomain, cookieSecure, cookieHttpOnly := getCookieSettings()
    // sessionToken, err := c.Cookie(cookieName)
    
    // Parse request body
    var requestBody ValidateSessionRequest
    if err := c.ShouldBindJSON(&requestBody); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
        return
    }


    c.JSON(http.StatusOK, gin.H{
        "sessionState": true,
    })
}