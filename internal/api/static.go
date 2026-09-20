package api

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:web_dist
var webFS embed.FS

// serveStaticFiles registers a catch-all handler that serves the embedded
// web UI assets. Exact file matches (JS, CSS, images) are served directly.
// All other non-API paths fall back to index.html for SPA client-side routing.
func (s *Server) serveStaticFiles() {
	// Strip the "web_dist" prefix so files are served from root
	sub, err := fs.Sub(webFS, "web_dist")
	if err != nil {
		// Embedded FS is compile-time; this can only fail if the path is wrong.
		panic("embedded web_dist: " + err.Error())
	}

	fileServer := http.FileServer(http.FS(sub))

	s.mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Let API routes pass through (they're registered with higher priority)
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}

		// Check if the file exists in the embedded FS
		cleanPath := strings.TrimPrefix(r.URL.Path, "/")
		if f, err := sub.Open(cleanPath); err == nil {
			_ = f.Close() // existence probe only; the file server reopens it
			fileServer.ServeHTTP(w, r)
			return
		}

		// SPA fallback: serve index.html for client-side routing
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	}))
}
