// Package middleware provides HTTP middleware for the API server.
package middleware

import (
	"crypto/subtle"
	"net/http"

	"github.com/gin-gonic/gin"
)

// BasicAuth creates a middleware that requires HTTP Basic Authentication.
// If username or password is empty, the middleware allows all requests through.
//
// Parameters:
//   - username: Required username (empty = no auth)
//   - password: Required password (empty = no auth)
//
// Returns:
//   - gin.HandlerFunc: Middleware function
func BasicAuth(username, password string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// If no credentials configured, allow access
		if username == "" || password == "" {
			c.Next()
			return
		}

		// Get credentials from request
		user, pass, ok := c.Request.BasicAuth()

		// Check if auth was provided
		if !ok {
			c.Header("WWW-Authenticate", `Basic realm="QuantumSpring Metrics"`)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "authentication required",
			})
			return
		}

		// Constant-time comparison to prevent timing attacks
		usernameMatch := subtle.ConstantTimeCompare([]byte(user), []byte(username)) == 1
		passwordMatch := subtle.ConstantTimeCompare([]byte(pass), []byte(password)) == 1

		if !usernameMatch || !passwordMatch {
			c.Header("WWW-Authenticate", `Basic realm="QuantumSpring Metrics"`)
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "invalid credentials",
			})
			return
		}

		// Authentication successful
		c.Next()
	}
}

// LocalhostOnly creates a middleware that only allows requests from localhost.
//
// Parameters:
//   - allowRemote: If true, allows requests from any IP
//
// Returns:
//   - gin.HandlerFunc: Middleware function
func LocalhostOnly(allowRemote bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		if allowRemote {
			c.Next()
			return
		}

		// Check if request is from localhost
		clientIP := c.ClientIP()
		if clientIP != "127.0.0.1" && clientIP != "::1" && clientIP != "localhost" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": "access denied: metrics API is bound to localhost only",
			})
			return
		}

		c.Next()
	}
}
