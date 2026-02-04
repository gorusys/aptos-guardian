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
			// Serve index.html without redirect; FileServer redirects /index.html -> / causing a loop
			f, err := fs.Open("index.html")
			if err != nil {
				http.NotFound(w, r)
				return
			}
			defer f.Close()
			stat, err := f.Stat()
			if err != nil || stat.IsDir() {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			http.ServeContent(w, r, "index.html", stat.ModTime(), f)
			return
		}
		path = filepath.Clean(path)
		if len(path) > 0 && path[0] != '/' {
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
