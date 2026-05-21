package version

import (
	"encoding/json"
	"net/http"
	"strings"

	localapi "github.com/ambient-code/platform/components/ambient-api-server/pkg/api"
	pkgserver "github.com/openshift-online/rh-trex-ai/pkg/server"
)

const versionPath = "/api/ambient/v1/version"

func init() {
	pkgserver.RegisterPreAuthMiddleware(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet && strings.TrimSuffix(r.URL.Path, "/") == versionPath {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(versionResponse{
					Version:   localapi.Version,
					BuildTime: localapi.BuildTime,
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	})
}

type versionResponse struct {
	Version   string `json:"version"`
	BuildTime string `json:"build_time"`
}
