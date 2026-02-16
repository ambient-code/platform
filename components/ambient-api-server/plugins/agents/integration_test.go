package agents_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	. "github.com/onsi/gomega"
	"gopkg.in/resty.v1"

	"github.com/ambient/platform/components/ambient-api-server/plugins/agents"
	"github.com/ambient/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/ambient/platform/components/ambient-api-server/test"
	"github.com/openshift-online/rh-trex-ai/pkg/environments"
)

func TestAgentGet(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	_, _, err := client.DefaultAPI.ApiAmbientApiServerV1AgentsIdGet(context.Background(), "foo").Execute()
	Expect(err).To(HaveOccurred(), "Expected 401 but got nil error")

	_, resp, err := client.DefaultAPI.ApiAmbientApiServerV1AgentsIdGet(ctx, "foo").Execute()
	Expect(err).To(HaveOccurred(), "Expected 404")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	agentModel, err := newAgent(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	agentOutput, resp, err := client.DefaultAPI.ApiAmbientApiServerV1AgentsIdGet(ctx, agentModel.ID).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	Expect(*agentOutput.Id).To(Equal(agentModel.ID), "found object does not match test object")
	Expect(*agentOutput.Kind).To(Equal("Agent"))
	Expect(*agentOutput.Href).To(Equal(fmt.Sprintf("/api/ambient-api-server/v1/agents/%s", agentModel.ID)))
	Expect(*agentOutput.CreatedAt).To(BeTemporally("~", agentModel.CreatedAt))
	Expect(*agentOutput.UpdatedAt).To(BeTemporally("~", agentModel.UpdatedAt))
}

func TestAgentPost(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	agentInput := openapi.Agent{
		Name:    "test-name",
		RepoUrl: openapi.PtrString("test-repo_url"),
		Prompt:  openapi.PtrString("test-prompt"),
	}

	agentOutput, resp, err := client.DefaultAPI.ApiAmbientApiServerV1AgentsPost(ctx).Agent(agentInput).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))
	Expect(*agentOutput.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(*agentOutput.Kind).To(Equal("Agent"))
	Expect(*agentOutput.Href).To(Equal(fmt.Sprintf("/api/ambient-api-server/v1/agents/%s", *agentOutput.Id)))

	jwtToken := ctx.Value(openapi.ContextAccessToken)
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		SetBody(`{ this is invalid }`).
		Post(h.RestURL("/agents"))

	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest))
}

func TestAgentPatch(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	agentModel, err := newAgent(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	agentOutput, resp, err := client.DefaultAPI.ApiAmbientApiServerV1AgentsIdPatch(ctx, agentModel.ID).AgentPatchRequest(openapi.AgentPatchRequest{}).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(*agentOutput.Id).To(Equal(agentModel.ID))
	Expect(*agentOutput.CreatedAt).To(BeTemporally("~", agentModel.CreatedAt))
	Expect(*agentOutput.Kind).To(Equal("Agent"))
	Expect(*agentOutput.Href).To(Equal(fmt.Sprintf("/api/ambient-api-server/v1/agents/%s", *agentOutput.Id)))

	jwtToken := ctx.Value(openapi.ContextAccessToken)
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		SetBody(`{ this is invalid }`).
		Patch(h.RestURL("/agents/foo"))

	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest))
}

func TestAgentPaging(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	_, err := newAgentList("Bronto", 20)
	Expect(err).NotTo(HaveOccurred())

	list, _, err := client.DefaultAPI.ApiAmbientApiServerV1AgentsGet(ctx).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting agent list: %v", err)
	Expect(len(list.Items)).To(Equal(20))
	Expect(list.Size).To(Equal(int32(20)))
	Expect(list.Total).To(Equal(int32(20)))
	Expect(list.Page).To(Equal(int32(1)))

	list, _, err = client.DefaultAPI.ApiAmbientApiServerV1AgentsGet(ctx).Page(2).Size(5).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting agent list: %v", err)
	Expect(len(list.Items)).To(Equal(5))
	Expect(list.Size).To(Equal(int32(5)))
	Expect(list.Total).To(Equal(int32(20)))
	Expect(list.Page).To(Equal(int32(2)))
}

func TestAgentAll(t *testing.T) {
	h, _ := test.RegisterIntegration(t)
	_ = h

	created, err := newAgentList("all", 5)
	Expect(err).NotTo(HaveOccurred())

	agentService := agents.Service(&environments.Environment().Services)
	all, svcErr := agentService.All(context.Background())
	Expect(svcErr).NotTo(HaveOccurred(), "Error calling All(): %v", svcErr)
	Expect(len(all)).To(Equal(5))

	returnedIDs := map[string]bool{}
	for _, a := range all {
		returnedIDs[a.ID] = true
	}
	for _, a := range created {
		Expect(returnedIDs).To(HaveKey(a.ID), "All() should return agent %s", a.ID)
	}
}

func TestAgentAllEmpty(t *testing.T) {
	h, _ := test.RegisterIntegration(t)
	_ = h

	agentService := agents.Service(&environments.Environment().Services)
	all, svcErr := agentService.All(context.Background())
	Expect(svcErr).NotTo(HaveOccurred(), "Error calling All() on empty table: %v", svcErr)
	Expect(len(all)).To(Equal(0))
}

func TestAgentListSearch(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	agents, err := newAgentList("bronto", 20)
	Expect(err).NotTo(HaveOccurred())

	search := fmt.Sprintf("id in ('%s')", agents[0].ID)
	list, _, err := client.DefaultAPI.ApiAmbientApiServerV1AgentsGet(ctx).Search(search).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting agent list: %v", err)
	Expect(len(list.Items)).To(Equal(1))
	Expect(list.Total).To(Equal(int32(1)))
	Expect(*list.Items[0].Id).To(Equal(agents[0].ID))
}
