package version

import (
	"encoding/json"
	"net/http"
	"strings"

	localapi "github.com/ambient-code/platform/components/ambient-api-server/pkg/api"
	pkgserver "github.com/openshift-online/rh-trex-ai/pkg/server"
)

const versionPath = "/api/ambient/v1/version"

var responseBytes []byte

func init() {
	responseBytes, _ = json.Marshal(versionResponse{
		Version:   localapi.Version,
		BuildTime: localapi.BuildTime,
		GitTag:    localapi.GitTag,
	})

	pkgserver.RegisterPreAuthMiddleware(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet && strings.TrimSuffix(r.URL.Path, "/") == versionPath {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write(responseBytes)
				return
			}
			next.ServeHTTP(w, r)
		})
	})
}

type versionResponse struct {
	Version   string `json:"version"`
	BuildTime string `json:"build_time"`
	GitTag    string `json:"git_tag"`
}
