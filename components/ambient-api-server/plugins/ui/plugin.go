package ui

import (
	"embed"
	"mime"
	"net/http"
	"path"
	"strings"

	pkgserver "github.com/openshift-online/rh-trex-ai/pkg/server"
)

//go:embed project-home.html
var projectHomeHTML []byte

//go:embed static
var staticFiles embed.FS

func init() {
	pkgserver.RegisterPreAuthMiddleware(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/ui/ambient/static/") {
				filePath := strings.TrimPrefix(r.URL.Path, "/ui/ambient/")
				data, err := staticFiles.ReadFile(filePath)
				if err != nil {
					http.Error(w, "not found", http.StatusNotFound)
					return
				}
				ext := path.Ext(filePath)
				ct := mime.TypeByExtension(ext)
				if ct == "" {
					ct = "application/octet-stream"
				}
				w.Header().Set("Content-Type", ct)
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(data)
				return
			}
			if strings.HasPrefix(r.URL.Path, "/ui/ambient") {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(projectHomeHTML)
				return
			}
			next.ServeHTTP(w, r)
		})
	})
}
