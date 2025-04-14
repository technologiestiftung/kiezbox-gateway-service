package handlers

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Cookie settings defaults
const (
    defaultCookieName     = "session_token"
    defaultCookieMaxAge   = 7200
    defaultCookiePath     = "/"
    defaultCookieDomain   = ""
    defaultCookieSecure   = false
    defaultCookieHttpOnly = true
)

// generateSessionToken creates a random session token
func generateSessionToken() string {
    bytes := make([]byte, 16)
    if _, err := rand.Read(bytes); err != nil {
        return time.Now().String()
    }
    return hex.EncodeToString(bytes)
}

// getCookieSettings retrieves cookie settings from environment variables
func getCookieSettings() (name string, maxAge int, path string, domain string, secure bool, httpOnly bool) {
    name = os.Getenv("COOKIE_NAME")
    if name == "" {
        name = defaultCookieName
    }

    maxAgeStr := os.Getenv("COOKIE_MAX_AGE")
    maxAge = defaultCookieMaxAge
    if maxAgeStr != "" {
        if val, err := strconv.Atoi(maxAgeStr); err == nil {
            maxAge = val
        }
    }

    path = os.Getenv("COOKIE_PATH")
    if path == "" {
        path = defaultCookiePath
    }

    domain = os.Getenv("COOKIE_DOMAIN")
    if domain == "" {
        domain = defaultCookieDomain
    }

    secureStr := os.Getenv("COOKIE_SECURE")
    secure = defaultCookieSecure
    if secureStr == "true" || secureStr == "1" {
        secure = true
    }

    // Get cookie httpOnly flag from env or use default
    httpOnlyStr := os.Getenv("COOKIE_HTTP_ONLY")
    httpOnly = defaultCookieHttpOnly
    if httpOnlyStr == "false" || httpOnlyStr == "0" {
        httpOnly = false
    }

    return
}

// @Summary Get Session
// @Description Returns session with session token, creating a new one only if needed
// @Tags session
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Configuration with session token"
// @Router /session [get]
func Session(c *gin.Context) {
    cookieName, cookieMaxAge, cookiePath, cookieDomain, cookieSecure, cookieHttpOnly := getCookieSettings()

    sessionToken, err := c.Cookie(cookieName)

    if sessionToken != "" {
        log.Printf("Session token found: %s", sessionToken)
    } else {
        log.Println("No session token found, generating a new one.")
    }

    if err != nil || sessionToken == "" {
        sessionToken = generateSessionToken()
        c.SetCookie(
            cookieName,
            sessionToken,
            cookieMaxAge,
            cookiePath,
            cookieDomain,
            cookieSecure,
            cookieHttpOnly,
        )
    }

    kiezboxConfig := gin.H{
        "kbServerAddress": os.Getenv("KB_SERVER_ADDRESS"),
        "kbWSSPort":       getIntEnv("KB_WSS_PORT", 8089),
        "kbWSSPath":       os.Getenv("KB_WSS_PATH"),
        "kbDomain":        os.Getenv("KB_DOMAIN"),
        "kbSIPUsername":   os.Getenv("KB_SIP_USERNAME"),
        "kbSIPPassword":   os.Getenv("KB_SIP_PASSWORD"),
        "kbisplayName":    os.Getenv("KB_DISPLAY_NAME"),
        "createdAt":       time.Now().Format(time.RFC3339),
        "updatedAt":       time.Now().Format(time.RFC3339),
    }

    c.JSON(http.StatusOK, gin.H{
        "config": kiezboxConfig,
    })
}

// Helper function to get integer environment variables with fallback
func getIntEnv(key string, fallback int) int {
    value := os.Getenv(key)
    if value == "" {
        return fallback
    }
    intValue, err := strconv.Atoi(value)
    if err != nil {
        return fallback
    }
    return intValue
}