package handlers

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	mathRand "math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Cookie settings defaults
const (
	defaultCookieName     = "session_token"
	defaultCookieMaxAge   = 86400
	defaultCookiePath     = "/"
	defaultCookieDomain   = ""
	defaultCookieSecure   = false
	defaultCookieHttpOnly = true
	defaultSessionPath    = ".kb-session"
	defaultUserPrefix     = "user"
)

type sipSession struct {
	Extension int64  `json:"extension"`
	Password  string `json:"password"`
	Timestamp int64  `json:"timestamp"`
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

func generatePassword() (string, error) {
	// Generate random bytes
	randomBytes := make([]byte, 24)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}
	base64Password := base64.URLEncoding.EncodeToString(randomBytes)
	return base64Password, nil
}

// Session handles requests related to user sessions.
// The behavior of the function depends on the HTTP method used:
// - `GET` and `HEAD`:
//   - Retrieves session information if a valid session token is provided in the cookie.
//   - For `GET`, it returns the session data as JSON.
//   - For `HEAD`, it returns only the status code (can be used to validate if a session is still valid)
//   - Returns a 401 Unauthorized if no session token is provided or a 404 Not Found if the session cannot be found.
//
// - `DELETE`:
//   - Deletes the session file associated with the provided cookie/token.
//   - Returns a 200 OK if the session is successfully deleted or a 500 Internal Server Error if the deletion fails.
//
// - `POST`:
//   - Creates a new session, generates a unique session token, and stores it
//   - This method will clean up the old session if a valid session cookie is provided with the request
//   - Ensures that extensions used in the session are unique, cleaning up expired sessions and checking for conflicts.
//   - Returns a 200 OK with the new session information if successful or a 503 Service Unavailable if no free sessions are available.
//
// - Other HTTP methods:
//   - Returns a 405 Method Not Allowed if the method is not supported.
//
// TODO: Adapt this to be 'real' openAPI doc?
func Session(c *gin.Context) {
	cookieName, cookieMaxAge, cookiePath, cookieDomain, cookieSecure, cookieHttpOnly := getCookieSettings()
	files, err := os.ReadDir(defaultSessionPath)
	if err != nil {
		slog.Error("Failed to read from session directory", "dir", defaultSessionPath, "err", err)
		c.String(http.StatusInternalServerError, "Failed to read from session directory %s: %v", defaultSessionPath, err)
		return
	}
	method := c.Request.Method
	slog.Info("Handling request", "method", method)
	if !(method == "GET" || method == "HEAD" || method == "DELETE" || method == "POST") {
		c.String(http.StatusMethodNotAllowed, "Method %s not allowed", method)
		return
	}
	sessionToken, _ := c.Cookie(cookieName)
	if sessionToken != "" {
		filePath := filepath.Join(defaultSessionPath, filepath.Clean(sessionToken)+".json")
		// Delete (old) session on DELETE or when the user is requesting a new one via POST
		if method == "DELETE" || method == "POST" {
			slog.Info("Removing session file", "file", filePath, "reason", method)
			err = os.Remove(filePath)
			if method == "DELETE" {
				if err != nil {
					slog.Error("Failed to remove session", "file", filePath, "err", err)
					//INFO: giving back the session token here (as part of the error message) defeats httponly cookies, keep that in mind
					c.String(http.StatusInternalServerError, "Failed to remove session:", err)
				} else {
					c.Status(http.StatusOK)
				}
				return
			}
		} else { // Retrieve sessions on GET or HEAD
			session_content, err := os.ReadFile(filePath)
			//TODO: also check for session expiration on GET/HEAD request
			if err != nil {
				//INFO: giving back the session token here defeats httponly cookies, keep that in mind
				if method == "GET" {
					c.String(http.StatusNotFound, "Failed to find session for token: %s", sessionToken)
				} else {
					c.Status(http.StatusNotFound)
				}
				return
			} else {
				if method == "GET" {
					c.Data(http.StatusOK, "application/json", session_content)
					return
				} else { //Only status code for HEAD requests
					c.Status(http.StatusOK)
					//TODO: HEAD request reported content legth is one off the actual length, see if this is a problem
					c.Header("Content-Length", strconv.Itoa(len(session_content)))
					//TODO: chack if we need some more 'manual' connection closing like
					//c.Header("Connection", "close")
					//c.Writer.Flush()
					//c.Abort()
					return
				}
			}
		}
	} else if method != "POST" {
		c.String(http.StatusUnauthorized, "No session token was provided")
		return
	}
	if method == "POST" {
		// An [int]bool map is kind of duplication, as existance of an extension can be tracked by a map without the value
		taken := make(map[int64]bool)
		// check for extensions already taken and clean up old sessions
		for _, file := range files {
			filePath := filepath.Join(defaultSessionPath, file.Name())
			if !file.IsDir() && strings.HasSuffix(file.Name(), ".json") {
				session_content, err := os.ReadFile(filePath)
				if err != nil {
					slog.Error("Failed to read file", "file", filePath, "err", err)
					continue
				}
				var session sipSession
				err = json.Unmarshal(session_content, &session)
				if err != nil {
					slog.Error("Error unmarshaling JSON", "err", err)
					continue
				}
				// removing timed out sessions while we are at it
				if time.Now().Unix() > defaultCookieMaxAge+session.Timestamp {
					slog.Info("Removing session file due to timeout", "file", filePath, "timestamp", session.Timestamp)
					err = os.Remove(filePath)
					continue
				}
				// checking if extension was already taken by another session
				//TODO: this should not happen? but currently is possible with race condition from multiple POSTs?
				if is_taken, exists := taken[session.Extension]; exists {
					if is_taken {
						slog.Info("Removing session file due to duplicate extension", "file", filePath, "extension", session.Extension)
						err = os.Remove(filePath)
						continue
					}
				} else {
					taken[session.Extension] = true
				}
			} else {
				slog.Info("Ignored file", "file", filePath)
			}
		}
		var new_session sipSession
		found_free := false
		//TODO: un-hardcode 1000 extension limit
		offs := mathRand.Intn(1000)
		for i := 0; i < 1000; i++ {
			var idx int64 = int64((offs + i) % 1000)
			if _, exists := taken[idx]; !exists {
				new_session.Extension = idx
				found_free = true
				break
			}
		}
		if found_free {
			newSessionToken := uuid.New().String()
			password, err := generatePassword()
			if err != nil {
				slog.Error("Failed to generate secure password", "err", err)
				c.String(http.StatusInternalServerError, "Failed to generate sercure Password:", err)
				return

			} else {
				new_session.Password = password
				new_session.Timestamp = time.Now().Unix()
				filePath := filepath.Join(defaultSessionPath, newSessionToken+".json")
				file, err := os.Create(filePath)
				if err != nil {
					slog.Error("Error creating file", "file", filePath, "err", err)
					c.String(http.StatusInternalServerError, "Error creating file:", err)
					return
				}
				defer file.Close()
				// Writes session content to file
				encoder := json.NewEncoder(file)
				err = encoder.Encode(new_session)
				if err != nil {
					slog.Error("Error encoding JSON", "err", err)
					return
				}
				c.SetCookie(
					cookieName,
					newSessionToken,
					cookieMaxAge,
					cookiePath,
					cookieDomain,
					cookieSecure,
					cookieHttpOnly,
				)
				c.JSON(http.StatusOK, new_session)
			}
		} else {
			c.String(http.StatusServiceUnavailable, "No more free sessions available")
			return
		}
	}
}

// TODO: move environment variables to some global state
// and maybe have them retrieved from some central system/uci config
func SIPConfig(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"kbServerAddress": os.Getenv("KB_SERVER_ADDRESS"),
		"kbWSSPort":       getIntEnv("KB_WSS_PORT", 8089),
		"kbWSSPath":       os.Getenv("KB_WSS_PATH"),
		"kbDomain":        os.Getenv("KB_DOMAIN"),
		"kbUserPrefix":    defaultUserPrefix,
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
