package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/sessions"
)

var Store *sessions.CookieStore

var serviceToken string
var serviceUserID string

func InitSession(secret string) {
	Store = sessions.NewCookieStore([]byte(secret))
	Store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 30,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

// Logger logs request method, path, status and latency.
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		latency := time.Since(start)
		gin.DefaultWriter.Write([]byte(
			time.Now().Format("2006/01/02 - 15:04:05") + " | " +
				http.StatusText(c.Writer.Status()) + " | " +
				latency.String() + " | " +
				c.Request.Method + " " + c.Request.URL.Path + "\n",
		))
	}
}

// Session loads the session and makes it available on the context.
func Session() gin.HandlerFunc {
	return func(c *gin.Context) {
		session, _ := Store.Get(c.Request, "pathfinder-session")
		c.Set("session", session)
		c.Next()
	}
}

// InitServiceToken stores the service token and its mapped user ID.
// Call once at startup before registering routes.
func InitServiceToken(token, userID string) {
	serviceToken = token
	serviceUserID = userID
}

// RequireAuth checks session for user_id and aborts with 401 if missing.
// Sets "user_id" (string) in the gin context for downstream handlers.
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		session, err := Store.Get(c.Request, "pathfinder-session")
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Not authenticated"})
			c.Abort()
			return
		}
		userIDVal, ok := session.Values["user_id"]
		if !ok || userIDVal == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "message": "Not authenticated"})
			c.Abort()
			return
		}
		c.Set("user_id", fmt.Sprintf("%v", userIDVal))
		c.Next()
	}
}

// ServiceTokenAuth validates the Bearer token in the Authorization header.
// On success it sets "user_id" to the configured serviceUserID in the gin context.
// Use this middleware on routes that must be accessible by machine callers
// (OpenClaw, curl) but not browser sessions.
func ServiceTokenAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		token, found := strings.CutPrefix(authHeader, "Bearer ")
		if !found || token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "service token required"})
			c.Abort()
			return
		}
		if token != serviceToken {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid service token"})
			c.Abort()
			return
		}
		c.Set("user_id", serviceUserID)
		c.Next()
	}
}
