package web

import (
	"io/fs"
	"net/http"
	"strings"
)

// Handler returns an http.Handler that serves the embedded SPA.
// It serves static files from the embedded build directory, and falls back
// to index.html for any unmatched paths (SPA client-side routing).
func Handler() http.Handler {
	sub, err := fs.Sub(buildFS, "build")
	if err != nil {
		panic("web: failed to create sub filesystem: " + err.Error())
	}

	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// API paths should never reach here
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}

		// Try to serve the file directly
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}

		// Check if the file exists in the embedded FS
		cleanPath := strings.TrimPrefix(path, "/")
		if f, err := sub.Open(cleanPath); err == nil {
			f.Close()
			fileServer.ServeHTTP(w, r)
			return
		}

		// Clean URLs (e.g. /browse) map to prerendered pages (browse.html).
		if cleanPath != "" && !strings.Contains(cleanPath, ".") {
			htmlPath := cleanPath + ".html"
			if f, err := sub.Open(htmlPath); err == nil {
				f.Close()
				r.URL.Path = "/" + htmlPath
				fileServer.ServeHTTP(w, r)
				return
			}
		}

		// SPA fallback: serve index.html for client-side routing
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}
