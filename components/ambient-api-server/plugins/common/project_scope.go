// Package common provides shared helpers for api-server plugin handlers.
package common

import (
	"fmt"
	"net/http"
	"regexp"

	"github.com/openshift-online/rh-trex-ai/pkg/errors"
	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

var safeProjectIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

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
	if !safeProjectIDPattern.MatchString(projectID) {
		return errors.Validation("invalid project_id format")
	}
	projectFilter := fmt.Sprintf("project_id = '%s'", projectID)
	if listArgs.Search != "" {
		listArgs.Search = fmt.Sprintf("%s and (%s)", projectFilter, listArgs.Search)
	} else {
		listArgs.Search = projectFilter
	}
	return nil
}
