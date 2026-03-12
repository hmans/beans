package commands

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestCSPMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(cspMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	t.Run("sets Content-Security-Policy header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		csp := rec.Header().Get("Content-Security-Policy")
		if csp == "" {
			t.Fatal("Content-Security-Policy header not set")
		}

		// Verify each directive is present
		directives := []string{
			"default-src 'self'",
			"script-src 'self'",
			"style-src 'self' 'unsafe-inline'",
			"img-src 'self' data: blob:",
			"font-src 'self'",
			"connect-src 'self' ws://localhost:* wss://localhost:* ws://127.0.0.1:* wss://127.0.0.1:*",
			"worker-src 'self' blob:",
			"object-src 'none'",
			"base-uri 'self'",
		}
		for _, d := range directives {
			if !strings.Contains(csp, d) {
				t.Errorf("CSP header missing directive %q\ngot: %s", d, csp)
			}
		}
	})

	t.Run("header is present on every response", func(t *testing.T) {
		// Request a route that doesn't exist
		req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)

		csp := rec.Header().Get("Content-Security-Policy")
		if csp == "" {
			t.Fatal("Content-Security-Policy header not set on 404 response")
		}
	})
}
