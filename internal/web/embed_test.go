package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandler(t *testing.T) {
	handler := Handler()

	t.Run("serves index.html for root path", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		// Should serve index.html (200) or at least not 500
		if rec.Code == http.StatusInternalServerError {
			t.Errorf("Handler returned 500 for root path")
		}
	})

	t.Run("returns 404 for missing asset files", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/nonexistent.js", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Errorf("Handler returned %d for missing asset, want 404", rec.Code)
		}
	})

	t.Run("serves index.html for SPA routes without extension", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/some/deep/route", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		// SPA routes should get index.html (200 or redirect), not 404
		if rec.Code == http.StatusNotFound {
			t.Errorf("Handler returned 404 for SPA route, should serve index.html")
		}
	})
}

func TestDistFS(t *testing.T) {
	t.Run("returns filesystem rooted at dist", func(t *testing.T) {
		fsys, err := DistFS()
		if err != nil {
			t.Fatalf("DistFS() error = %v", err)
		}
		if fsys == nil {
			t.Error("DistFS() returned nil filesystem")
		}
	})
}
