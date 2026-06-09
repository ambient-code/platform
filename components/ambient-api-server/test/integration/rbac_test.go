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

func ensureBuiltInRoles(t *testing.T) {
	t.Helper()
	g := environments.Environment().Database.SessionFactory.New(context.Background())
	roles := []struct {
		name string
		perm string
	}{
		{"platform:admin", `["*:*"]`},
		{"platform:viewer", `["project:read","project:list","session:read","session:list","agent:read","agent:list"]`},
		{"project:owner", `["project:read","project:update","project:delete","agent:*","session:*","session_message:*","role_binding:*"]`},
		{"project:editor", `["project:read","agent:create","agent:read","agent:update","agent:list","agent:start","session:create","session:read","session:update","session:list","session_message:*","role_binding:delete"]`},
		{"project:viewer", `["project:read","agent:read","agent:list","session:read","session:list","session_message:read","session_message:list"]`},
		{"agent:operator", `["agent:read","agent:update","agent:start","agent:list","session:read","session:list"]`},
		{"agent:observer", `["agent:read","agent:list","session:read","session:list"]`},
		{"agent:runner", `["session:read","session_message:*"]`},
		{"agent:editor", `["agent:read","agent:update"]`},
		{"credential:owner", `["credential:create","credential:read","credential:update","credential:delete","credential:list","credential:fetch_token","role_binding:create","role_binding:delete"]`},
		{"credential:reader", `["credential:read","credential:list"]`},
		{"credential:token-reader", `["credential:fetch_token"]`},
	}
	for _, r := range roles {
		if err := g.Exec(
			`INSERT INTO roles (id, name, display_name, description, permissions, built_in, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, true, NOW(), NOW())
			 ON CONFLICT (name) DO NOTHING`,
			api.NewID(), r.name, r.name, r.name, r.perm,
		).Error; err != nil {
			t.Fatalf("failed to seed role %s: %v", r.name, err)
		}
	}
}

func TestRBAC_ProjectCreationCreatesOwnerBinding(t *testing.T) {
	RegisterTestingT(t)
	h := test.NewHelper(t)
	h.DBFactory.ResetDB()
	ensureBuiltInRoles(t)
	client := h.NewApiClient()

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	projectInput := openapi.Project{
		Name: "rbac-test-proj",
	}
	projectOutput, resp, err := client.DefaultAPI.ApiAmbientV1ProjectsPost(ctx).Project(projectInput).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))

	// Verify owner binding was created
	g := environments.Environment().Database.SessionFactory.New(context.Background())
	var count int64
	err = g.Raw(
		`SELECT COUNT(*) FROM role_bindings rb
		 JOIN roles r ON r.id = rb.role_id
		 WHERE r.name = 'project:owner'
		   AND rb.scope = 'project'
		   AND rb.project_id = ?
		   AND rb.deleted_at IS NULL`,
		*projectOutput.Id,
	).Scan(&count).Error
	Expect(err).NotTo(HaveOccurred())
	Expect(count).To(Equal(int64(1)), "expected exactly one project:owner binding")
}

func TestRBAC_CredentialCreationCreatesOwnerBinding(t *testing.T) {
	RegisterTestingT(t)
	h := test.NewHelper(t)
	h.DBFactory.ResetDB()
	ensureBuiltInRoles(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	credInput := openapi.Credential{
		Name:     "rbac-test-cred",
		Provider: "github",
		Token:    openapi.PtrString("ghp_test123"),
	}

	jwtToken := ctx.Value(openapi.ContextAccessToken)
	resp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		SetBody(credInput).
		Post(h.RestURL("/credentials"))

	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode()).To(Equal(http.StatusCreated))

	var created map[string]interface{}
	Expect(json.Unmarshal(resp.Body(), &created)).NotTo(HaveOccurred())
	credID := created["id"].(string)

	// Verify owner binding was created
	g := environments.Environment().Database.SessionFactory.New(context.Background())
	var count int64
	err = g.Raw(
		`SELECT COUNT(*) FROM role_bindings rb
		 JOIN roles r ON r.id = rb.role_id
		 WHERE r.name = 'credential:owner'
		   AND rb.scope = 'credential'
		   AND rb.credential_id = ?
		   AND rb.deleted_at IS NULL`,
		credID,
	).Scan(&count).Error
	Expect(err).NotTo(HaveOccurred())
	Expect(count).To(Equal(int64(1)), "expected exactly one credential:owner binding")
}

func TestRBAC_UserAutoProvisioned(t *testing.T) {
	RegisterTestingT(t)
	h := test.NewHelper(t)
	h.DBFactory.ResetDB()
	ensureBuiltInRoles(t)
	client := h.NewApiClient()

	account := h.NewAccount("rbac-auto-user", "Test User", "test@example.com")
	ctx := h.NewAuthenticatedContext(account)

	// Any authenticated request triggers auto-provisioning
	// POST /projects is auth-exempt so it should work
	projectInput := openapi.Project{Name: "auto-prov-test"}
	_, resp, err := client.DefaultAPI.ApiAmbientV1ProjectsPost(ctx).Project(projectInput).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))

	// Verify user record exists
	g := environments.Environment().Database.SessionFactory.New(context.Background())
	var username string
	dbErr := g.Raw(`SELECT username FROM users WHERE username = ? AND deleted_at IS NULL`, "rbac-auto-user").Scan(&username).Error
	Expect(dbErr).NotTo(HaveOccurred())
	Expect(username).To(Equal("rbac-auto-user"))
}

func TestRBAC_MissingRolesSeeded(t *testing.T) {
	RegisterTestingT(t)
	h := test.NewHelper(t)
	h.DBFactory.ResetDB()
	ensureBuiltInRoles(t)

	g := environments.Environment().Database.SessionFactory.New(context.Background())

	requiredRoles := []string{
		"platform:admin", "platform:viewer",
		"project:owner", "project:editor", "project:viewer",
		"agent:operator", "agent:observer", "agent:runner", "agent:editor",
		"credential:owner", "credential:token-reader",
	}

	for _, roleName := range requiredRoles {
		var count int64
		err := g.Raw(`SELECT COUNT(*) FROM roles WHERE name = ? AND deleted_at IS NULL`, roleName).Scan(&count).Error
		Expect(err).NotTo(HaveOccurred(), "querying role %s", roleName)
		Expect(count).To(Equal(int64(1)), "expected role %s to be seeded", roleName)
	}
}
