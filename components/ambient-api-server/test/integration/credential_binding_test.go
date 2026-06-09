package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	. "github.com/onsi/gomega"
	"gopkg.in/resty.v1"

	"github.com/ambient-code/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/ambient-code/platform/components/ambient-api-server/test"
	"github.com/openshift-online/rh-trex-ai/pkg/api"
	"github.com/openshift-online/rh-trex-ai/pkg/environments"
)

// setupCredentialBindingTest seeds roles, creates a credential owned by ownerUsername,
// and returns (credentialID, credentialRoleID for "credential:owner" scope bindings).
// The caller is responsible for calling h.DBFactory.ResetDB() first.
func setupCredentialBindingTest(t *testing.T, h *test.Helper, ownerUsername string) (credID string, credOwnerRoleID string) {
	t.Helper()
	ensureBuiltInRoles(t)
	g := environments.Environment().Database.SessionFactory.New(context.Background())

	// Create a credential directly in the DB
	credID = api.NewID()
	err := g.Exec(
		`INSERT INTO credentials (id, name, provider, token, created_at, updated_at)
		 VALUES (?, ?, ?, ?, NOW(), NOW())`,
		credID, "test-cred", "github", "encrypted-token",
	).Error
	Expect(err).NotTo(HaveOccurred())

	// Look up credential:owner role ID
	err = g.Raw(`SELECT id FROM roles WHERE name = 'credential:owner' AND deleted_at IS NULL`).Scan(&credOwnerRoleID).Error
	Expect(err).NotTo(HaveOccurred())
	Expect(credOwnerRoleID).NotTo(BeEmpty())

	// Create credential:owner binding for ownerUsername
	err = g.Exec(
		`INSERT INTO role_bindings (id, role_id, scope, user_id, credential_id, created_at, updated_at)
		 VALUES (?, ?, 'credential', ?, ?, NOW(), NOW())`,
		api.NewID(), credOwnerRoleID, ownerUsername, credID,
	).Error
	Expect(err).NotTo(HaveOccurred())

	return credID, credOwnerRoleID
}

// setupProjectWithRole creates a project and gives username the specified role on it.
// Returns the project ID and the role ID used.
func setupProjectWithRole(t *testing.T, username, roleName string) (projectID string, roleID string) {
	t.Helper()
	g := environments.Environment().Database.SessionFactory.New(context.Background())

	projectID = api.NewID()
	err := g.Exec(
		`INSERT INTO projects (id, name, created_at, updated_at)
		 VALUES (?, ?, NOW(), NOW())`,
		projectID, fmt.Sprintf("proj-%s", projectID[:8]),
	).Error
	Expect(err).NotTo(HaveOccurred())

	err = g.Raw(`SELECT id FROM roles WHERE name = ? AND deleted_at IS NULL`, roleName).Scan(&roleID).Error
	Expect(err).NotTo(HaveOccurred())
	Expect(roleID).NotTo(BeEmpty(), "role %s not found", roleName)

	err = g.Exec(
		`INSERT INTO role_bindings (id, role_id, scope, user_id, project_id, created_at, updated_at)
		 VALUES (?, ?, 'project', ?, ?, NOW(), NOW())`,
		api.NewID(), roleID, username, projectID,
	).Error
	Expect(err).NotTo(HaveOccurred())

	return projectID, roleID
}

// createCredentialBinding creates a credential-scope role binding via the REST API using resty.
func createCredentialBinding(h *test.Helper, ctx context.Context, body map[string]interface{}) (*resty.Response, error) {
	jwtToken := ctx.Value(openapi.ContextAccessToken)
	return resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		SetBody(body).
		Post(h.RestURL("/role_bindings"))
}

// deleteBinding deletes a role binding via the REST API using resty.
func deleteBinding(h *test.Helper, ctx context.Context, bindingID string) (*resty.Response, error) {
	jwtToken := ctx.Value(openapi.ContextAccessToken)
	return resty.R().
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		Delete(h.RestURL(fmt.Sprintf("/role_bindings/%s", bindingID)))
}

// --- Credential Binding Create Tests ---

func TestCredentialBinding_ProjectEditorCanBind(t *testing.T) {
	RegisterTestingT(t)
	h := test.NewHelper(t)
	h.DBFactory.ResetDB()

	username := "editor-user"
	account := h.NewAccount(username, "Editor User", "editor@test.com")
	ctx := h.NewAuthenticatedContext(account)

	credID, _ := setupCredentialBindingTest(t, h, username)
	projectID, _ := setupProjectWithRole(t, username, "project:editor")

	// Look up a role to bind (credential:reader for example)
	g := environments.Environment().Database.SessionFactory.New(context.Background())
	var credReaderRoleID string
	err := g.Raw(`SELECT id FROM roles WHERE name = 'credential:reader' AND deleted_at IS NULL`).Scan(&credReaderRoleID).Error
	Expect(err).NotTo(HaveOccurred())

	resp, err := createCredentialBinding(h, ctx, map[string]interface{}{
		"role_id":       credReaderRoleID,
		"scope":         "credential",
		"credential_id": credID,
		"project_id":    projectID,
		"user_id":       "some-other-user",
	})
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode()).To(Equal(http.StatusCreated),
		"project:editor should be able to bind credentials; got %d: %s", resp.StatusCode(), resp.String())
}

func TestCredentialBinding_ProjectViewerCannotBind(t *testing.T) {
	RegisterTestingT(t)
	h := test.NewHelper(t)
	h.DBFactory.ResetDB()

	username := "viewer-user"
	account := h.NewAccount(username, "Viewer User", "viewer@test.com")
	ctx := h.NewAuthenticatedContext(account)

	credID, _ := setupCredentialBindingTest(t, h, username)
	projectID, _ := setupProjectWithRole(t, username, "project:viewer")

	g := environments.Environment().Database.SessionFactory.New(context.Background())
	var credReaderRoleID string
	err := g.Raw(`SELECT id FROM roles WHERE name = 'credential:reader' AND deleted_at IS NULL`).Scan(&credReaderRoleID).Error
	Expect(err).NotTo(HaveOccurred())

	resp, err := createCredentialBinding(h, ctx, map[string]interface{}{
		"role_id":       credReaderRoleID,
		"scope":         "credential",
		"credential_id": credID,
		"project_id":    projectID,
		"user_id":       "some-other-user",
	})
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode()).To(Equal(http.StatusForbidden),
		"project:viewer should NOT be able to bind credentials; got %d: %s", resp.StatusCode(), resp.String())
}

func TestCredentialBinding_AgentWithoutProjectRejected(t *testing.T) {
	RegisterTestingT(t)
	h := test.NewHelper(t)
	h.DBFactory.ResetDB()

	username := "admin-user"
	account := h.NewAccount(username, "Admin User", "admin@test.com")
	ctx := h.NewAuthenticatedContext(account)

	credID, _ := setupCredentialBindingTest(t, h, username)

	// Give user platform:admin so they'd pass all other checks
	g := environments.Environment().Database.SessionFactory.New(context.Background())
	var adminRoleID string
	err := g.Raw(`SELECT id FROM roles WHERE name = 'platform:admin' AND deleted_at IS NULL`).Scan(&adminRoleID).Error
	Expect(err).NotTo(HaveOccurred())
	err = g.Exec(
		`INSERT INTO role_bindings (id, role_id, scope, user_id, created_at, updated_at)
		 VALUES (?, ?, 'global', ?, NOW(), NOW())`,
		api.NewID(), adminRoleID, username,
	).Error
	Expect(err).NotTo(HaveOccurred())

	var credReaderRoleID string
	err = g.Raw(`SELECT id FROM roles WHERE name = 'credential:reader' AND deleted_at IS NULL`).Scan(&credReaderRoleID).Error
	Expect(err).NotTo(HaveOccurred())

	// agent_id without project_id should be rejected
	resp, err := createCredentialBinding(h, ctx, map[string]interface{}{
		"role_id":       credReaderRoleID,
		"scope":         "credential",
		"credential_id": credID,
		"agent_id":      "some-agent-id",
		// project_id intentionally omitted
	})
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode()).To(Equal(http.StatusBadRequest),
		"agent_id without project_id should return 400; got %d: %s", resp.StatusCode(), resp.String())
}

func TestCredentialBinding_AgentNotInProjectRejected(t *testing.T) {
	RegisterTestingT(t)
	h := test.NewHelper(t)
	h.DBFactory.ResetDB()

	username := "owner-user"
	account := h.NewAccount(username, "Owner User", "owner@test.com")
	ctx := h.NewAuthenticatedContext(account)

	credID, _ := setupCredentialBindingTest(t, h, username)
	projectID, _ := setupProjectWithRole(t, username, "project:owner")

	// Create an agent in a DIFFERENT project
	g := environments.Environment().Database.SessionFactory.New(context.Background())
	otherProjectID := api.NewID()
	err := g.Exec(
		`INSERT INTO projects (id, name, created_at, updated_at)
		 VALUES (?, ?, NOW(), NOW())`,
		otherProjectID, "other-project",
	).Error
	Expect(err).NotTo(HaveOccurred())

	agentID := api.NewID()
	err = g.Exec(
		`INSERT INTO agents (id, project_id, name, created_at, updated_at)
		 VALUES (?, ?, ?, NOW(), NOW())`,
		agentID, otherProjectID, "wrong-project-agent",
	).Error
	Expect(err).NotTo(HaveOccurred())

	var credReaderRoleID string
	err = g.Raw(`SELECT id FROM roles WHERE name = 'credential:reader' AND deleted_at IS NULL`).Scan(&credReaderRoleID).Error
	Expect(err).NotTo(HaveOccurred())

	// Binding agent from otherProject to projectID should fail
	resp, err := createCredentialBinding(h, ctx, map[string]interface{}{
		"role_id":       credReaderRoleID,
		"scope":         "credential",
		"credential_id": credID,
		"project_id":    projectID,
		"agent_id":      agentID,
	})
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode()).To(Equal(http.StatusBadRequest),
		"agent not in project should return 400; got %d: %s", resp.StatusCode(), resp.String())
}

func TestCredentialBinding_GlobalRequiresAdmin(t *testing.T) {
	RegisterTestingT(t)
	h := test.NewHelper(t)
	h.DBFactory.ResetDB()

	username := "nonadmin-user"
	account := h.NewAccount(username, "Non-Admin User", "nonadmin@test.com")
	ctx := h.NewAuthenticatedContext(account)

	credID, _ := setupCredentialBindingTest(t, h, username)

	g := environments.Environment().Database.SessionFactory.New(context.Background())
	var credReaderRoleID string
	err := g.Raw(`SELECT id FROM roles WHERE name = 'credential:reader' AND deleted_at IS NULL`).Scan(&credReaderRoleID).Error
	Expect(err).NotTo(HaveOccurred())

	// Global binding (no project_id, no agent_id) without platform:admin
	resp, err := createCredentialBinding(h, ctx, map[string]interface{}{
		"role_id":       credReaderRoleID,
		"scope":         "credential",
		"credential_id": credID,
		// project_id and agent_id intentionally omitted → global binding
	})
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode()).To(Equal(http.StatusForbidden),
		"non-admin should not create global credential binding; got %d: %s", resp.StatusCode(), resp.String())
}

func TestCredentialBinding_NonCredentialOwnerCannotBind(t *testing.T) {
	RegisterTestingT(t)
	h := test.NewHelper(t)
	h.DBFactory.ResetDB()

	ownerUsername := "cred-owner"
	nonOwnerUsername := "non-owner"
	ensureBuiltInRoles(t)

	// Create credential owned by ownerUsername
	g := environments.Environment().Database.SessionFactory.New(context.Background())
	credID := api.NewID()
	err := g.Exec(
		`INSERT INTO credentials (id, name, provider, token, created_at, updated_at)
		 VALUES (?, ?, ?, ?, NOW(), NOW())`,
		credID, "someone-elses-cred", "github", "encrypted-token",
	).Error
	Expect(err).NotTo(HaveOccurred())

	var credOwnerRoleID string
	err = g.Raw(`SELECT id FROM roles WHERE name = 'credential:owner' AND deleted_at IS NULL`).Scan(&credOwnerRoleID).Error
	Expect(err).NotTo(HaveOccurred())
	err = g.Exec(
		`INSERT INTO role_bindings (id, role_id, scope, user_id, credential_id, created_at, updated_at)
		 VALUES (?, ?, 'credential', ?, ?, NOW(), NOW())`,
		api.NewID(), credOwnerRoleID, ownerUsername, credID,
	).Error
	Expect(err).NotTo(HaveOccurred())

	// Give non-owner project:owner on a project
	projectID, _ := setupProjectWithRole(t, nonOwnerUsername, "project:owner")

	account := h.NewAccount(nonOwnerUsername, "Non Owner", "nonowner@test.com")
	ctx := h.NewAuthenticatedContext(account)

	var credReaderRoleID string
	err = g.Raw(`SELECT id FROM roles WHERE name = 'credential:reader' AND deleted_at IS NULL`).Scan(&credReaderRoleID).Error
	Expect(err).NotTo(HaveOccurred())

	resp, err := createCredentialBinding(h, ctx, map[string]interface{}{
		"role_id":       credReaderRoleID,
		"scope":         "credential",
		"credential_id": credID,
		"project_id":    projectID,
	})
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode()).To(Equal(http.StatusForbidden),
		"non-credential-owner should not bind; got %d: %s", resp.StatusCode(), resp.String())
}

// --- Credential Binding Delete Tests ---

func TestCredentialBinding_ProjectEditorCanUnbindWithoutCredentialOwner(t *testing.T) {
	RegisterTestingT(t)
	h := test.NewHelper(t)
	h.DBFactory.ResetDB()

	credOwnerUsername := "cred-owner"
	editorUsername := "project-editor"

	credID, _ := setupCredentialBindingTest(t, h, credOwnerUsername)
	projectID, _ := setupProjectWithRole(t, editorUsername, "project:editor")

	// Create a credential-scope binding on the project (simulating cred-owner bound it)
	g := environments.Environment().Database.SessionFactory.New(context.Background())
	var credReaderRoleID string
	err := g.Raw(`SELECT id FROM roles WHERE name = 'credential:reader' AND deleted_at IS NULL`).Scan(&credReaderRoleID).Error
	Expect(err).NotTo(HaveOccurred())

	bindingID := api.NewID()
	err = g.Exec(
		`INSERT INTO role_bindings (id, role_id, scope, user_id, credential_id, project_id, created_at, updated_at)
		 VALUES (?, ?, 'credential', ?, ?, ?, NOW(), NOW())`,
		bindingID, credReaderRoleID, "some-user", credID, projectID,
	).Error
	Expect(err).NotTo(HaveOccurred())

	// Editor (not credential owner) should be able to delete binding from their project
	account := h.NewAccount(editorUsername, "Editor User", "editor@test.com")
	ctx := h.NewAuthenticatedContext(account)

	resp, err := deleteBinding(h, ctx, bindingID)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode()).To(Equal(http.StatusNoContent),
		"project:editor should unbind credentials without credential:owner; got %d: %s", resp.StatusCode(), resp.String())
}

func TestCredentialBinding_ProjectViewerCannotUnbind(t *testing.T) {
	RegisterTestingT(t)
	h := test.NewHelper(t)
	h.DBFactory.ResetDB()

	credOwnerUsername := "cred-owner"
	viewerUsername := "project-viewer"

	credID, _ := setupCredentialBindingTest(t, h, credOwnerUsername)
	projectID, _ := setupProjectWithRole(t, viewerUsername, "project:viewer")

	g := environments.Environment().Database.SessionFactory.New(context.Background())
	var credReaderRoleID string
	err := g.Raw(`SELECT id FROM roles WHERE name = 'credential:reader' AND deleted_at IS NULL`).Scan(&credReaderRoleID).Error
	Expect(err).NotTo(HaveOccurred())

	bindingID := api.NewID()
	err = g.Exec(
		`INSERT INTO role_bindings (id, role_id, scope, user_id, credential_id, project_id, created_at, updated_at)
		 VALUES (?, ?, 'credential', ?, ?, ?, NOW(), NOW())`,
		bindingID, credReaderRoleID, "some-user", credID, projectID,
	).Error
	Expect(err).NotTo(HaveOccurred())

	account := h.NewAccount(viewerUsername, "Viewer User", "viewer@test.com")
	ctx := h.NewAuthenticatedContext(account)

	resp, err := deleteBinding(h, ctx, bindingID)
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode()).To(Equal(http.StatusForbidden),
		"project:viewer should NOT unbind credentials; got %d: %s", resp.StatusCode(), resp.String())
}

// --- Service Account / Internal Role Tests ---

func TestCredentialBinding_ServiceAccountCanCreateInternalRole(t *testing.T) {
	RegisterTestingT(t)
	h := test.NewHelper(t)
	h.DBFactory.ResetDB()
	ensureBuiltInRoles(t)

	// Simulate a service account by giving the user platform:admin.
	// In the real system, middleware.IsServiceCaller checks a context flag set
	// during auth. In integration tests, the mock auth middleware may or may not
	// set this flag. If this test fails with "cannot assign internal role", that's
	// the expected RED state — the implementation must allow platform:admin callers
	// (service accounts) to create internal role bindings.
	username := "service-account-cp"
	account := h.NewAccount(username, "Control Plane SA", "cp@svc.local")
	ctx := h.NewAuthenticatedContext(account)

	g := environments.Environment().Database.SessionFactory.New(context.Background())

	// Give user platform:admin
	var adminRoleID string
	err := g.Raw(`SELECT id FROM roles WHERE name = 'platform:admin' AND deleted_at IS NULL`).Scan(&adminRoleID).Error
	Expect(err).NotTo(HaveOccurred())
	err = g.Exec(
		`INSERT INTO role_bindings (id, role_id, scope, user_id, created_at, updated_at)
		 VALUES (?, ?, 'global', ?, NOW(), NOW())`,
		api.NewID(), adminRoleID, username,
	).Error
	Expect(err).NotTo(HaveOccurred())

	// Create a credential
	credID := api.NewID()
	err = g.Exec(
		`INSERT INTO credentials (id, name, provider, token, created_at, updated_at)
		 VALUES (?, ?, ?, ?, NOW(), NOW())`,
		credID, "sa-test-cred", "github", "encrypted",
	).Error
	Expect(err).NotTo(HaveOccurred())

	// credential:owner binding for the SA
	var credOwnerRoleID string
	err = g.Raw(`SELECT id FROM roles WHERE name = 'credential:owner' AND deleted_at IS NULL`).Scan(&credOwnerRoleID).Error
	Expect(err).NotTo(HaveOccurred())
	err = g.Exec(
		`INSERT INTO role_bindings (id, role_id, scope, user_id, credential_id, created_at, updated_at)
		 VALUES (?, ?, 'credential', ?, ?, NOW(), NOW())`,
		api.NewID(), credOwnerRoleID, username, credID,
	).Error
	Expect(err).NotTo(HaveOccurred())

	// Try to create credential:token-reader binding (internal role)
	var tokenReaderRoleID string
	err = g.Raw(`SELECT id FROM roles WHERE name = 'credential:token-reader' AND deleted_at IS NULL`).Scan(&tokenReaderRoleID).Error
	Expect(err).NotTo(HaveOccurred())

	resp, err := createCredentialBinding(h, ctx, map[string]interface{}{
		"role_id":       tokenReaderRoleID,
		"scope":         "credential",
		"credential_id": credID,
		"user_id":       username,
	})
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode()).To(Equal(http.StatusCreated),
		"platform:admin (service account) should create internal role bindings; got %d: %s", resp.StatusCode(), resp.String())
}

// --- Valid Agent-Level Binding Test ---

func TestCredentialBinding_AgentInProjectAccepted(t *testing.T) {
	RegisterTestingT(t)
	h := test.NewHelper(t)
	h.DBFactory.ResetDB()

	username := "owner-user"
	account := h.NewAccount(username, "Owner User", "owner@test.com")
	ctx := h.NewAuthenticatedContext(account)

	credID, _ := setupCredentialBindingTest(t, h, username)
	projectID, _ := setupProjectWithRole(t, username, "project:owner")

	// Create an agent IN this project
	g := environments.Environment().Database.SessionFactory.New(context.Background())
	agentID := api.NewID()
	err := g.Exec(
		`INSERT INTO agents (id, project_id, name, created_at, updated_at)
		 VALUES (?, ?, ?, NOW(), NOW())`,
		agentID, projectID, "correct-project-agent",
	).Error
	Expect(err).NotTo(HaveOccurred())

	var credReaderRoleID string
	err = g.Raw(`SELECT id FROM roles WHERE name = 'credential:reader' AND deleted_at IS NULL`).Scan(&credReaderRoleID).Error
	Expect(err).NotTo(HaveOccurred())

	resp, err := createCredentialBinding(h, ctx, map[string]interface{}{
		"role_id":       credReaderRoleID,
		"scope":         "credential",
		"credential_id": credID,
		"project_id":    projectID,
		"agent_id":      agentID,
	})
	Expect(err).NotTo(HaveOccurred())

	// Parse the response to check the result
	var body map[string]interface{}
	_ = json.Unmarshal(resp.Body(), &body)

	Expect(resp.StatusCode()).To(Equal(http.StatusCreated),
		"agent-level binding with correct project should succeed; got %d: %s", resp.StatusCode(), resp.String())
}
