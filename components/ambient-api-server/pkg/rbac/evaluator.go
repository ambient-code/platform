package rbac

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/openshift-online/rh-trex-ai/pkg/db"
	"gorm.io/gorm"
)

type bindingRow struct {
	RoleID       string
	RoleName     string
	Scope        string
	UserID       *string
	ProjectID    *string
	AgentID      *string
	SessionID    *string
	CredentialID *string
	Permissions  string
}

type Evaluator struct {
	sessionFactory *db.SessionFactory
}

func NewEvaluator(sessionFactory *db.SessionFactory) *Evaluator {
	return &Evaluator{sessionFactory: sessionFactory}
}

func (e *Evaluator) fetchBindings(g *gorm.DB, username string) ([]bindingRow, error) {
	var rows []bindingRow
	err := g.Table("role_bindings rb").
		Select("rb.role_id, r.name AS role_name, rb.scope, rb.user_id, rb.project_id, rb.agent_id, rb.session_id, rb.credential_id, r.permissions").
		Joins("JOIN roles r ON r.id = rb.role_id").
		Where("rb.user_id = ? AND rb.deleted_at IS NULL AND r.deleted_at IS NULL", username).
		Scan(&rows).Error
	return rows, err
}

func (e *Evaluator) Evaluate(ctx context.Context, username string, resource Resource, action Action, scope RequestScope) (bool, error) {
	g := (*e.sessionFactory).New(ctx)

	bindings, err := e.fetchBindings(g, username)
	if err != nil {
		return false, err
	}
	if len(bindings) == 0 {
		return false, nil
	}

	requiredPerm := string(resource) + ":" + string(action)

	if scope.SessionID != "" && scope.ProjectID == "" {
		projectID, lookupErr := e.resolveSessionProject(g, scope.SessionID)
		if lookupErr == nil && projectID != "" {
			scope.ProjectID = projectID
		}
	}

	for _, b := range bindings {
		if !bindingMatchesPermission(b.Permissions, requiredPerm) {
			continue
		}
		if bindingCoversScope(b, scope) {
			return true, nil
		}
	}

	return false, nil
}

func (e *Evaluator) AuthorizedProjectIDs(ctx context.Context, username string) (projectIDs []string, isGlobal bool, err error) {
	g := (*e.sessionFactory).New(ctx)

	bindings, fetchErr := e.fetchBindings(g, username)
	if fetchErr != nil {
		return nil, false, fetchErr
	}

	seen := map[string]bool{}
	for _, b := range bindings {
		if b.Scope == string(ScopeGlobal) {
			return nil, true, nil
		}
		if b.Scope == string(ScopeProject) && b.ProjectID != nil {
			if !seen[*b.ProjectID] {
				seen[*b.ProjectID] = true
				projectIDs = append(projectIDs, *b.ProjectID)
			}
		}
	}
	return projectIDs, false, nil
}

func (e *Evaluator) AuthorizedCredentialIDs(ctx context.Context, username string) (credentialIDs []string, isGlobal bool, err error) {
	g := (*e.sessionFactory).New(ctx)

	bindings, fetchErr := e.fetchBindings(g, username)
	if fetchErr != nil {
		return nil, false, fetchErr
	}

	seen := map[string]bool{}
	for _, b := range bindings {
		if b.Scope == string(ScopeGlobal) {
			return nil, true, nil
		}
		if b.Scope == string(ScopeCredential) && b.CredentialID != nil {
			if !seen[*b.CredentialID] {
				seen[*b.CredentialID] = true
				credentialIDs = append(credentialIDs, *b.CredentialID)
			}
		}
	}
	return credentialIDs, false, nil
}

func (e *Evaluator) resolveSessionProject(g *gorm.DB, sessionID string) (string, error) {
	var projectID string
	err := g.Table("sessions").
		Select("COALESCE(project_id, '')").
		Where("id = ? AND deleted_at IS NULL", sessionID).
		Scan(&projectID).Error
	return projectID, err
}

func bindingMatchesPermission(permissionsJSON, required string) bool {
	var perms []string
	if err := json.Unmarshal([]byte(permissionsJSON), &perms); err != nil {
		return false
	}

	reqParts := strings.SplitN(required, ":", 2)
	if len(reqParts) != 2 {
		return false
	}
	reqResource, reqAction := reqParts[0], reqParts[1]

	for _, perm := range perms {
		if perm == "*:*" {
			return true
		}
		parts := strings.SplitN(perm, ":", 2)
		if len(parts) != 2 {
			continue
		}
		r, a := parts[0], parts[1]
		resourceMatch := r == "*" || r == reqResource
		actionMatch := a == "*" || a == reqAction
		if resourceMatch && actionMatch {
			return true
		}
	}
	return false
}

func bindingCoversScope(b bindingRow, reqScope RequestScope) bool {
	switch ScopeLevel(b.Scope) {
	case ScopeGlobal:
		return true

	case ScopeProject:
		if b.ProjectID == nil {
			return false
		}
		return reqScope.ProjectID == *b.ProjectID

	case ScopeAgent:
		if b.AgentID == nil {
			return false
		}
		return reqScope.AgentID == *b.AgentID

	case ScopeSession:
		if b.SessionID == nil {
			return false
		}
		return reqScope.SessionID == *b.SessionID

	case ScopeCredential:
		if b.CredentialID == nil {
			return false
		}
		return reqScope.CredentialID == *b.CredentialID

	default:
		return false
	}
}
