package api

import (
	"net/http"
	"os"
	"path/filepath"
)

func StaticHandler(webRoot string) http.Handler {
	if webRoot == "" {
		return http.NotFoundHandler()
	}
	fs := http.Dir(webRoot)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		path := r.URL.Path
		if path == "/" || path == "" {
			path = "/index.html"
		}
		path = filepath.Clean(path)
		if path[0] != '/' {
			path = "/" + path
		}
		r.URL.Path = path
		http.FileServer(fs).ServeHTTP(w, r)
	})
}

func DefaultWebRoot() string {
	dir := "web"
	if _, err := os.Stat(dir); err == nil {
		return dir
	}
	return ""
}
