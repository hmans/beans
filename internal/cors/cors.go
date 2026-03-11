package cors

import (
	"net/http"
	"strings"
)

// DefaultOrigins are the default allowed origins for CORS and WebSocket.
// They cover common local development scenarios.
var DefaultOrigins = []string{
	"http://localhost:*",
	"http://127.0.0.1:*",
}

// Checker validates HTTP origins against an allowlist.
// Patterns support a trailing `:*` wildcard to match any port.
type Checker struct {
	allowAll bool
	patterns []pattern
}

type pattern struct {
	prefix   string // scheme + host portion before port wildcard
	exact    string // non-empty for exact matches
	wildcard bool   // true if port is wildcarded
}

// NewChecker creates a Checker from a list of origin patterns.
// Supported patterns:
//   - "*" allows all origins
//   - "http://localhost:5173" exact match
//   - "http://localhost:*" matches any port on localhost
func NewChecker(origins []string) *Checker {
	c := &Checker{}
	for _, o := range origins {
		if o == "*" {
			c.allowAll = true
			return c
		}
		c.patterns = append(c.patterns, parsePattern(o))
	}
	return c
}

func parsePattern(s string) pattern {
	if strings.HasSuffix(s, ":*") {
		return pattern{
			prefix:   s[:len(s)-1], // keep "scheme://host:" prefix
			wildcard: true,
		}
	}
	return pattern{exact: s}
}

// IsAllowed returns true if the given origin is allowed.
func (c *Checker) IsAllowed(origin string) bool {
	if c.allowAll {
		return true
	}
	if origin == "" {
		// Same-origin requests (no Origin header) are always allowed
		return true
	}
	for _, p := range c.patterns {
		if p.wildcard {
			if strings.HasPrefix(origin, p.prefix) {
				port := origin[len(p.prefix):]
				if port != "" && isDigits(port) {
					return true
				}
			}
		} else if p.exact == origin {
			return true
		}
	}
	return false
}

// CheckOriginFunc returns a function suitable for gorilla/websocket's Upgrader.CheckOrigin.
func (c *Checker) CheckOriginFunc() func(r *http.Request) bool {
	return func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		return c.IsAllowed(origin)
	}
}

// CORSOrigin returns the value for the Access-Control-Allow-Origin header.
// If allowAll, returns "*". If the specific origin is allowed, returns it
// (enabling per-origin CORS). Returns empty string if not allowed.
func (c *Checker) CORSOrigin(origin string) string {
	if c.allowAll {
		return "*"
	}
	if c.IsAllowed(origin) && origin != "" {
		return origin
	}
	return ""
}

func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
