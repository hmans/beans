package web

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed dist/*
var distFS embed.FS

// DistFS returns the embedded frontend filesystem, rooted at dist/.
func DistFS() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}

// Handler returns an http.Handler that serves the embedded SPA.
// It handles SPA routing by serving index.html for paths that don't match a file.
func Handler() http.Handler {
	fsys, err := DistFS()
	if err != nil {
		// This shouldn't happen with embed, but handle gracefully
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "frontend not available", http.StatusInternalServerError)
		})
	}

	fileServer := http.FileServer(http.FS(fsys))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Clean the path
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}

		// Try to open the file
		path = strings.TrimPrefix(path, "/")
		f, err := fsys.Open(path)
		if err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		// File not found - serve index.html for SPA routing
		// But only for paths that look like routes (not missing assets)
		if !strings.Contains(path, ".") {
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
			return
		}

		// Actual missing file (e.g., missing .js, .css)
		http.NotFound(w, r)
	})
}
