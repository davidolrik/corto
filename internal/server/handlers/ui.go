package handlers

import (
	"io/fs"
	"net/http"
)

// RegisterUIRoutes serves the admin web UI from the given filesystem under
// /admin/. Paths that don't match a file fall back to index.html so the SPA
// can handle client-side routing.
func RegisterUIRoutes(mux *http.ServeMux, ui fs.FS) {
	fileServer := http.FileServerFS(ui)

	mux.Handle("GET /admin", http.RedirectHandler("/admin/", http.StatusMovedPermanently))
	mux.Handle("GET /admin/", http.StripPrefix("/admin/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "" {
			path = "index.html"
		}
		if _, err := fs.Stat(ui, path); err != nil {
			// Unknown path: serve the SPA shell for client-side routing
			http.ServeFileFS(w, r, ui, "index.html")
			return
		}
		fileServer.ServeHTTP(w, r)
	})))
}
