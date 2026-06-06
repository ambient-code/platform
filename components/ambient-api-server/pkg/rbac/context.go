package rbac

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/openshift-online/rh-trex-ai/pkg/services"
)

// safeTSLValuePattern matches values safe for interpolation into TSL search
// expressions (KSUIDs, usernames, etc.). Rejects SQL/TSL metacharacters.
var safeTSLValuePattern = regexp.MustCompile(`^[a-zA-Z0-9_.@:\-]+$`)

// ValidateTSLValues checks that every value in the slice is safe for
// interpolation into a TSL search string. Returns an error naming the
// first invalid value found.
func ValidateTSLValues(values []string) error {
	for _, v := range values {
		if !safeTSLValuePattern.MatchString(v) {
			return fmt.Errorf("unsafe value for TSL interpolation: %q", v)
		}
	}
	return nil
}

// --- TSL expression builders ---
// These helpers construct Tree Search Language (TSL) filter expressions
// without raw fmt.Sprintf, enforcing value validation at construction time.

// TSLEqual builds a TSL equality expression: column = 'value'.
// Returns an error if the value contains unsafe characters.
func TSLEqual(column, value string) (string, error) {
	if err := ValidateTSLValues([]string{value}); err != nil {
		return "", err
	}
	return column + " = '" + value + "'", nil
}

// TSLIn builds a TSL set-membership expression: column in ('v1','v2',...).
// Returns an error if any value contains unsafe characters or if values is empty.
func TSLIn(column string, values []string) (string, error) {
	if len(values) == 0 {
		return "", fmt.Errorf("TSLIn requires at least one value")
	}
	if err := ValidateTSLValues(values); err != nil {
		return "", err
	}
	quoted := make([]string, len(values))
	for i, v := range values {
		quoted[i] = "'" + v + "'"
	}
	return column + " in (" + strings.Join(quoted, ",") + ")", nil
}

// TSLAnd combines two non-empty TSL expressions with " and ".
// Empty operands are skipped; if both are empty, returns "".
func TSLAnd(a, b string) string {
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}
	return "(" + a + ") and (" + b + ")"
}

// TSLOr combines two non-empty TSL expressions with " or ".
// Empty operands are skipped; if both are empty, returns "".
func TSLOr(a, b string) string {
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}
	return a + " or " + b
}

// AppendTSLFilter merges a new filter into listArgs.Search with " and ".
func AppendTSLFilter(listArgs *services.ListArguments, filter string) {
	listArgs.Search = TSLAnd(listArgs.Search, filter)
}

// PrependTSLFilter merges a new filter before listArgs.Search with " and ".
func PrependTSLFilter(listArgs *services.ListArguments, filter string) {
	listArgs.Search = TSLAnd(filter, listArgs.Search)
}

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

	scopeFilter, err := TSLIn(filterColumn, ids)
	if err != nil {
		return false
	}

	AppendTSLFilter(listArgs, scopeFilter)
	return true
}
