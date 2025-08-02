package auth

import (
	"net/http"
	"strings"

	"github.com/jamesprial/nexus/internal/interfaces"
	"github.com/jamesprial/nexus/internal/utils"
)

// AuthMiddleware handles API key authentication and transformation
type AuthMiddleware struct {
	keyManager interfaces.KeyManager
	logger     interfaces.Logger
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(keyManager interfaces.KeyManager, logger interfaces.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		keyManager: keyManager,
		logger:     logger,
	}
}

// Middleware returns the HTTP middleware function
func (a *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract client API key from Authorization header
		authHeader := r.Header.Get("Authorization")
		clientKey := strings.TrimSpace(authHeader)
		
		// Remove "Bearer " prefix if present
		if strings.HasPrefix(clientKey, "Bearer ") {
			clientKey = strings.TrimSpace(clientKey[7:])
		}
		
		// Check if key is provided
		if clientKey == "" {
			if a.logger != nil {
				a.logger.Warn("Missing API key in request", map[string]any{
					"path":   r.URL.Path,
					"method": r.Method,
				})
			}
			http.Error(w, "Missing API key", http.StatusUnauthorized)
			return
		}
		
		// Validate client key
		if !a.keyManager.ValidateClientKey(clientKey) {
			if a.logger != nil {
				a.logger.Warn("Invalid client API key", map[string]any{
					"path":       r.URL.Path,
					"method":     r.Method,
					"client_key": utils.MaskAPIKey(clientKey),
				})
			}
			http.Error(w, "Invalid API key", http.StatusUnauthorized)
			return
		}
		
		// Get upstream key
		upstreamKey, err := a.keyManager.GetUpstreamKey(clientKey)
		if err != nil {
			if a.logger != nil {
				a.logger.Error("Failed to get upstream API key", map[string]any{
					"error":      err.Error(),
					"client_key": utils.MaskAPIKey(clientKey),
				})
			}
			http.Error(w, "Authentication failed", http.StatusUnauthorized)
			return
		}
		
		// Replace Authorization header with upstream key
		// Preserve Bearer prefix if it was in the original
		if strings.HasPrefix(authHeader, "Bearer ") {
			r.Header.Set("Authorization", "Bearer "+upstreamKey)
		} else {
			r.Header.Set("Authorization", upstreamKey)
		}
		
		if a.logger != nil {
			a.logger.Debug("API key authenticated and transformed", map[string]any{
				"path":         r.URL.Path,
				"method":       r.Method,
				"client_key":   utils.MaskAPIKey(clientKey),
				"upstream_key": utils.MaskAPIKey(upstreamKey),
			})
		}
		
		// Continue to next handler
		next.ServeHTTP(w, r)
	})
}

