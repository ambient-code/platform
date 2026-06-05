package rbac

import (
	"context"
	"fmt"
	"strings"

	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

type authResultKey struct{}

type AuthResult struct {
	Username      string
	IsGlobalAdmin bool
	ProjectIDs    []string // nil = global access (all projects)
	CredentialIDs []string // nil = global access (all credentials)
}

func SetAuthResult(ctx context.Context, result *AuthResult) context.Context {
	return context.WithValue(ctx, authResultKey{}, result)
}

func GetAuthResult(ctx context.Context) *AuthResult {
	v, _ := ctx.Value(authResultKey{}).(*AuthResult)
	return v
}

// ApplyListFilter restricts list results to the caller's authorized scope.
// filterColumn is the DB column to filter on (e.g. "id" for projects, "project_id" for sessions).
// useCredentialIDs controls whether to filter by credential IDs instead of project IDs.
// Returns false if the user has zero authorized IDs (caller should return empty list).
func ApplyListFilter(ctx context.Context, listArgs *services.ListArguments, filterColumn string, useCredentialIDs bool) bool {
	auth := GetAuthResult(ctx)
	if auth == nil {
		return false
	}
	if auth.IsGlobalAdmin {
		return true
	}

	var ids []string
	if useCredentialIDs {
		ids = auth.CredentialIDs
	} else {
		ids = auth.ProjectIDs
	}

	if len(ids) == 0 {
		return false
	}

	quoted := make([]string, len(ids))
	for i, id := range ids {
		quoted[i] = fmt.Sprintf("'%s'", strings.ReplaceAll(id, "'", "''"))
	}
	scopeFilter := fmt.Sprintf("%s in (%s)", filterColumn, strings.Join(quoted, ","))

	if listArgs.Search != "" {
		listArgs.Search = fmt.Sprintf("(%s) and (%s)", listArgs.Search, scopeFilter)
	} else {
		listArgs.Search = scopeFilter
	}
	return true
}
