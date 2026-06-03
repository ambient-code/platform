package agents_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/ambient-code/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/ambient-code/platform/components/ambient-api-server/plugins/projects"
	"github.com/ambient-code/platform/components/ambient-api-server/test"
	"github.com/openshift-online/rh-trex-ai/pkg/environments"
)

// newTestProject creates a project via the service layer for use in agent tests.
var testProjectCounter int

func newTestProject() (*projects.Project, error) {
	testProjectCounter++
	projectService := projects.Service(&environments.Environment().Services)
	project := &projects.Project{
		Name:        fmt.Sprintf("agent-test-proj-%d", testProjectCounter),
		Description: stringPtr("project for agent integration tests"),
		Status:      stringPtr("active"),
	}
	result, svcErr := projectService.Create(context.Background(), project)
	if svcErr != nil {
		return nil, fmt.Errorf("projects.Create: %s", svcErr.Error())
	}
	return result, nil
}

func TestAgentGet(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	// Unauthenticated request should fail
	_, _, err := client.DefaultAPI.ApiAmbientV1ProjectsIdAgentsAgentIdGet(context.Background(), "proj", "foo").Execute()
	Expect(err).To(HaveOccurred(), "Expected 401 but got nil error")

	// Create a project and agent
	proj, err := newTestProject()
	Expect(err).NotTo(HaveOccurred())

	agentModel, err := newAgentWithProject("get-test-agent", proj.ID)
	Expect(err).NotTo(HaveOccurred())

	// Not found for wrong agent ID
	_, resp, err := client.DefaultAPI.ApiAmbientV1ProjectsIdAgentsAgentIdGet(ctx, proj.ID, "nonexistent").Execute()
	Expect(err).To(HaveOccurred(), "Expected 404")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	// Successful GET
	agentOutput, resp, err := client.DefaultAPI.ApiAmbientV1ProjectsIdAgentsAgentIdGet(ctx, proj.ID, agentModel.ID).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	Expect(*agentOutput.Id).To(Equal(agentModel.ID), "found agent does not match test agent")
	Expect(*agentOutput.Kind).To(Equal("Agent"))
	Expect(*agentOutput.Href).To(Equal(fmt.Sprintf("/api/ambient/v1/agents/%s", agentModel.ID)))
	Expect(agentOutput.Name).To(Equal("get-test-agent"))
	Expect(agentOutput.ProjectId).To(Equal(proj.ID))
	Expect(*agentOutput.CreatedAt).To(BeTemporally("~", agentModel.CreatedAt))
	Expect(*agentOutput.UpdatedAt).To(BeTemporally("~", agentModel.UpdatedAt))

	// LLM defaults should be present
	Expect(agentOutput.LlmModel).NotTo(BeNil(), "llm_model should be defaulted")
	Expect(*agentOutput.LlmModel).To(Equal("claude-sonnet-4-6"))
	Expect(agentOutput.OwnerUserId).NotTo(BeNil(), "owner_user_id should be present")
}

func TestAgentPost(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	proj, err := newTestProject()
	Expect(err).NotTo(HaveOccurred())

	agentInput := openapi.Agent{
		ProjectId:      proj.ID,
		Name:           "post-test-agent",
		DisplayName:    openapi.PtrString("Post Test Agent"),
		Description:    openapi.PtrString("An agent for testing POST"),
		RepoUrl:        openapi.PtrString("https://github.com/test/repo"),
		LlmModel:       openapi.PtrString("claude-opus-4-20250514"),
		LlmTemperature: openapi.PtrFloat64(0.5),
		LlmMaxTokens:   openapi.PtrInt32(8192),
		Prompt:         openapi.PtrString("You are a test agent"),
		Labels:         openapi.PtrString(`{"env":"test"}`),
		Annotations:    openapi.PtrString(`{"team":"platform"}`),
	}

	agentOutput, resp, err := client.DefaultAPI.ApiAmbientV1ProjectsIdAgentsPost(ctx, proj.ID).Agent(agentInput).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error posting agent: %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))

	Expect(*agentOutput.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(*agentOutput.Kind).To(Equal("Agent"))
	Expect(*agentOutput.Href).To(Equal(fmt.Sprintf("/api/ambient/v1/agents/%s", *agentOutput.Id)))
	Expect(agentOutput.Name).To(Equal("post-test-agent"))
	Expect(agentOutput.ProjectId).To(Equal(proj.ID))
	Expect(*agentOutput.DisplayName).To(Equal("Post Test Agent"))
	Expect(*agentOutput.Description).To(Equal("An agent for testing POST"))
	Expect(*agentOutput.RepoUrl).To(Equal("https://github.com/test/repo"))
	Expect(*agentOutput.LlmModel).To(Equal("claude-opus-4-20250514"))
	Expect(*agentOutput.LlmTemperature).To(BeNumerically("~", 0.5, 0.001))
	Expect(*agentOutput.LlmMaxTokens).To(Equal(int32(8192)))
	Expect(*agentOutput.Prompt).To(Equal("You are a test agent"))
	Expect(*agentOutput.Labels).To(Equal(`{"env":"test"}`))
	Expect(*agentOutput.Annotations).To(Equal(`{"team":"platform"}`))

	// Verify the agent can be fetched
	fetched, resp, err := client.DefaultAPI.ApiAmbientV1ProjectsIdAgentsAgentIdGet(ctx, proj.ID, *agentOutput.Id).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(fetched.Name).To(Equal("post-test-agent"))
	Expect(*fetched.LlmModel).To(Equal("claude-opus-4-20250514"))
}

func TestAgentPostLlmDefaults(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	proj, err := newTestProject()
	Expect(err).NotTo(HaveOccurred())

	// Create agent without LLM fields — should get defaults
	agentInput := openapi.Agent{
		ProjectId: proj.ID,
		Name:      "default-llm-agent",
	}

	agentOutput, resp, err := client.DefaultAPI.ApiAmbientV1ProjectsIdAgentsPost(ctx, proj.ID).Agent(agentInput).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error posting agent: %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))

	Expect(agentOutput.LlmModel).NotTo(BeNil(), "llm_model should be defaulted")
	Expect(*agentOutput.LlmModel).To(Equal("claude-sonnet-4-6"))
	Expect(agentOutput.LlmTemperature).NotTo(BeNil(), "llm_temperature should be present")
	Expect(*agentOutput.LlmTemperature).To(BeNumerically("~", 0.0, 0.001))
	Expect(agentOutput.LlmMaxTokens).NotTo(BeNil(), "llm_max_tokens should be present")
	Expect(*agentOutput.LlmMaxTokens).To(Equal(int32(0)))
}

func TestAgentPatch(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	proj, err := newTestProject()
	Expect(err).NotTo(HaveOccurred())

	// Create agent with initial values
	agentInput := openapi.Agent{
		ProjectId:      proj.ID,
		Name:           "patch-test-agent",
		DisplayName:    openapi.PtrString("Original Name"),
		LlmModel:       openapi.PtrString("claude-sonnet-4-20250514"),
		LlmTemperature: openapi.PtrFloat64(0.7),
		RepoUrl:        openapi.PtrString("https://github.com/original/repo"),
		Description:    openapi.PtrString("Original description"),
	}

	created, resp, err := client.DefaultAPI.ApiAmbientV1ProjectsIdAgentsPost(ctx, proj.ID).Agent(agentInput).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))

	// Patch with new values
	patchReq := openapi.AgentPatchRequest{
		DisplayName: openapi.PtrString("Updated Name"),
		LlmModel:    openapi.PtrString("claude-opus-4-20250514"),
		RepoUrl:     openapi.PtrString("https://github.com/updated/repo"),
		Description: openapi.PtrString("Updated description"),
	}

	patched, resp, err := client.DefaultAPI.ApiAmbientV1ProjectsIdAgentsAgentIdPatch(ctx, proj.ID, *created.Id).AgentPatchRequest(patchReq).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error patching agent: %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	Expect(*patched.Id).To(Equal(*created.Id))
	Expect(*patched.DisplayName).To(Equal("Updated Name"))
	Expect(*patched.LlmModel).To(Equal("claude-opus-4-20250514"))
	Expect(*patched.RepoUrl).To(Equal("https://github.com/updated/repo"))
	Expect(*patched.Description).To(Equal("Updated description"))

	// Verify name was not changed (not in patch)
	Expect(patched.Name).To(Equal("patch-test-agent"))

	// Verify via GET
	fetched, resp, err := client.DefaultAPI.ApiAmbientV1ProjectsIdAgentsAgentIdGet(ctx, proj.ID, *created.Id).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(*fetched.DisplayName).To(Equal("Updated Name"))
	Expect(*fetched.LlmModel).To(Equal("claude-opus-4-20250514"))
}

func TestAgentPatchTemperatureZeroPreserved(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	proj, err := newTestProject()
	Expect(err).NotTo(HaveOccurred())

	// Create with non-zero temperature
	agentInput := openapi.Agent{
		ProjectId:      proj.ID,
		Name:           "zero-temp-agent",
		LlmTemperature: openapi.PtrFloat64(0.7),
	}
	created, resp, err := client.DefaultAPI.ApiAmbientV1ProjectsIdAgentsPost(ctx, proj.ID).Agent(agentInput).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))
	Expect(*created.LlmTemperature).To(BeNumerically("~", 0.7, 0.001))

	// Patch temperature to 0.0 — must not be overwritten by a default
	patchReq := openapi.AgentPatchRequest{
		LlmTemperature: openapi.PtrFloat64(0.0),
	}
	patched, resp, err := client.DefaultAPI.ApiAmbientV1ProjectsIdAgentsAgentIdPatch(ctx, proj.ID, *created.Id).AgentPatchRequest(patchReq).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(*patched.LlmTemperature).To(BeNumerically("~", 0.0, 0.001), "temperature 0.0 must be preserved, not overwritten")

	// Verify via GET
	fetched, resp, err := client.DefaultAPI.ApiAmbientV1ProjectsIdAgentsAgentIdGet(ctx, proj.ID, *created.Id).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(*fetched.LlmTemperature).To(BeNumerically("~", 0.0, 0.001), "temperature 0.0 must survive round-trip")
}

func TestAgentPaging(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	proj, err := newTestProject()
	Expect(err).NotTo(HaveOccurred())

	// Create 20 agents in this project
	for i := 1; i <= 20; i++ {
		_, createErr := newAgentWithProject(fmt.Sprintf("paging-agent-%d", i), proj.ID)
		Expect(createErr).NotTo(HaveOccurred())
	}

	// Default page: all 20
	list, _, err := client.DefaultAPI.ApiAmbientV1ProjectsIdAgentsGet(ctx, proj.ID).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting agent list: %v", err)
	Expect(len(list.Items)).To(Equal(20))
	Expect(list.Size).To(Equal(int32(20)))
	Expect(list.Total).To(Equal(int32(20)))
	Expect(list.Page).To(Equal(int32(1)))

	// Page 2, size 5
	list, _, err = client.DefaultAPI.ApiAmbientV1ProjectsIdAgentsGet(ctx, proj.ID).Page(2).Size(5).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting agent list page 2: %v", err)
	Expect(len(list.Items)).To(Equal(5))
	Expect(list.Size).To(Equal(int32(5)))
	Expect(list.Total).To(Equal(int32(20)))
	Expect(list.Page).To(Equal(int32(2)))
}

func TestAgentListSearch(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	proj, err := newTestProject()
	Expect(err).NotTo(HaveOccurred())

	// Create agents with distinct names
	agent1, err := newAgentWithProject("searchable-alpha", proj.ID)
	Expect(err).NotTo(HaveOccurred())
	_, err = newAgentWithProject("searchable-beta", proj.ID)
	Expect(err).NotTo(HaveOccurred())
	_, err = newAgentWithProject("other-gamma", proj.ID)
	Expect(err).NotTo(HaveOccurred())

	// Search by exact ID
	search := fmt.Sprintf("id in ('%s')", agent1.ID)
	list, _, err := client.DefaultAPI.ApiAmbientV1ProjectsIdAgentsGet(ctx, proj.ID).Search(search).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error searching agents: %v", err)
	Expect(len(list.Items)).To(Equal(1))
	Expect(list.Total).To(Equal(int32(1)))
	Expect(*list.Items[0].Id).To(Equal(agent1.ID))

	// Search by name pattern
	searchName := "name like 'searchable%'"
	list, _, err = client.DefaultAPI.ApiAmbientV1ProjectsIdAgentsGet(ctx, proj.ID).Search(searchName).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error searching agents by name: %v", err)
	Expect(len(list.Items)).To(Equal(2))
	Expect(list.Total).To(Equal(int32(2)))
}

func TestAgentDelete(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	proj, err := newTestProject()
	Expect(err).NotTo(HaveOccurred())

	agentInput := openapi.Agent{
		ProjectId: proj.ID,
		Name:      "delete-test-agent",
	}
	created, resp, err := client.DefaultAPI.ApiAmbientV1ProjectsIdAgentsPost(ctx, proj.ID).Agent(agentInput).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))

	// Delete the agent
	resp, err = client.DefaultAPI.ApiAmbientV1ProjectsIdAgentsAgentIdDelete(ctx, proj.ID, *created.Id).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error deleting agent: %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusNoContent))

	// Verify it's gone
	_, resp, err = client.DefaultAPI.ApiAmbientV1ProjectsIdAgentsAgentIdGet(ctx, proj.ID, *created.Id).Execute()
	Expect(err).To(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
}

func TestAgentCrossProjectIsolation(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	proj1, err := newTestProject()
	Expect(err).NotTo(HaveOccurred())
	proj2, err := newTestProject()
	Expect(err).NotTo(HaveOccurred())

	// Create agent in project 1
	agentInput := openapi.Agent{
		ProjectId: proj1.ID,
		Name:      "isolated-agent",
	}
	created, resp, err := client.DefaultAPI.ApiAmbientV1ProjectsIdAgentsPost(ctx, proj1.ID).Agent(agentInput).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))

	// GET via project 2 should fail (agent does not belong to project 2)
	_, resp, err = client.DefaultAPI.ApiAmbientV1ProjectsIdAgentsAgentIdGet(ctx, proj2.ID, *created.Id).Execute()
	Expect(err).To(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusForbidden))

	// List via project 2 should return empty
	list, _, err := client.DefaultAPI.ApiAmbientV1ProjectsIdAgentsGet(ctx, proj2.ID).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(len(list.Items)).To(Equal(0))
}
