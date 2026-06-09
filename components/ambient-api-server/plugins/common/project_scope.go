// Package common provides shared helpers for api-server plugin handlers.
package common

import (
	"net/http"

	pkgrbac "github.com/ambient-code/platform/components/ambient-api-server/pkg/rbac"
	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

// ApplyProjectScope reads the project ID from the query parameter or the
// X-Ambient-Project header (query param takes precedence) and injects a
// project_id filter into listArgs.Search. Returns a validation error if the
// project ID contains unsafe characters.
func ApplyProjectScope(r *http.Request, listArgs *services.ListArguments) *errors.ServiceError {
	projectID := r.URL.Query().Get("project_id")
	if projectID == "" {
		projectID = r.Header.Get("X-Ambient-Project")
	}
	if projectID == "" {
		return nil
	}
	projectFilter, err := pkgrbac.TSLEqual("project_id", projectID)
	if err != nil {
		return errors.Validation("invalid project_id format")
	}
	pkgrbac.PrependTSLFilter(listArgs, projectFilter)
	return nil
}
