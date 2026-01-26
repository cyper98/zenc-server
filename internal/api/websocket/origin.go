package websocket

import (
	"net/http"
	"strings"
)

// MakeOriginChecker creates an origin checker function from allowed origins list
func MakeOriginChecker(allowedOrigins []string) OriginChecker {
	// If no origins specified, allow all
	if len(allowedOrigins) == 0 {
		return func(r *http.Request) bool {
			return true
		}
	}

	return func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true // Allow requests with no origin (e.g., native apps)
		}

		for _, allowed := range allowedOrigins {
			if matchOrigin(origin, allowed) {
				return true
			}
		}

		return false
	}
}

// matchOrigin checks if origin matches the pattern
// Supports wildcards like "https://*.example.com" or "http://localhost:*"
func matchOrigin(origin, pattern string) bool {
	// Exact match
	if origin == pattern {
		return true
	}

	// Wildcard matching
	if strings.Contains(pattern, "*") {
		// Replace * with regex-like matching
		parts := strings.Split(pattern, "*")

		// Check if origin starts with first part
		if !strings.HasPrefix(origin, parts[0]) {
			return false
		}

		// Check if origin ends with last part (if exists)
		if len(parts) > 1 && parts[len(parts)-1] != "" {
			if !strings.HasSuffix(origin, parts[len(parts)-1]) {
				return false
			}
		}

		return true
	}

	return false
}
