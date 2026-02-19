package workflows_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	. "github.com/onsi/gomega"
	"gopkg.in/resty.v1"

	"github.com/ambient/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/ambient/platform/components/ambient-api-server/test"
)

func TestWorkflowGet(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	_, _, err := client.DefaultAPI.ApiAmbientApiServerV1WorkflowsIdGet(context.Background(), "foo").Execute()
	Expect(err).To(HaveOccurred(), "Expected 401 but got nil error")

	_, resp, err := client.DefaultAPI.ApiAmbientApiServerV1WorkflowsIdGet(ctx, "foo").Execute()
	Expect(err).To(HaveOccurred(), "Expected 404")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	workflowModel, err := newWorkflow(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	workflowOutput, resp, err := client.DefaultAPI.ApiAmbientApiServerV1WorkflowsIdGet(ctx, workflowModel.ID).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	Expect(*workflowOutput.Id).To(Equal(workflowModel.ID), "found object does not match test object")
	Expect(*workflowOutput.Kind).To(Equal("Workflow"))
	Expect(*workflowOutput.Href).To(Equal(fmt.Sprintf("/api/ambient-api-server/v1/workflows/%s", workflowModel.ID)))
	Expect(*workflowOutput.CreatedAt).To(BeTemporally("~", workflowModel.CreatedAt))
	Expect(*workflowOutput.UpdatedAt).To(BeTemporally("~", workflowModel.UpdatedAt))
}

func TestWorkflowPost(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	agent, err := newAgent()
	Expect(err).NotTo(HaveOccurred(), "Error creating parent agent")

	workflowInput := openapi.Workflow{
		Name:    "test-name",
		RepoUrl: openapi.PtrString("test-repo_url"),
		Prompt:  openapi.PtrString("test-prompt"),
		AgentId: openapi.PtrString(agent.ID),
	}

	workflowOutput, resp, err := client.DefaultAPI.ApiAmbientApiServerV1WorkflowsPost(ctx).Workflow(workflowInput).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))
	Expect(*workflowOutput.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(*workflowOutput.Kind).To(Equal("Workflow"))
	Expect(*workflowOutput.Href).To(Equal(fmt.Sprintf("/api/ambient-api-server/v1/workflows/%s", *workflowOutput.Id)))

	jwtToken := ctx.Value(openapi.ContextAccessToken)
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		SetBody(`{ this is invalid }`).
		Post(h.RestURL("/workflows"))

	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest))
}

func TestWorkflowPatch(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	workflowModel, err := newWorkflow(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	workflowOutput, resp, err := client.DefaultAPI.ApiAmbientApiServerV1WorkflowsIdPatch(ctx, workflowModel.ID).WorkflowPatchRequest(openapi.WorkflowPatchRequest{}).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(*workflowOutput.Id).To(Equal(workflowModel.ID))
	Expect(*workflowOutput.CreatedAt).To(BeTemporally("~", workflowModel.CreatedAt))
	Expect(*workflowOutput.Kind).To(Equal("Workflow"))
	Expect(*workflowOutput.Href).To(Equal(fmt.Sprintf("/api/ambient-api-server/v1/workflows/%s", *workflowOutput.Id)))

	jwtToken := ctx.Value(openapi.ContextAccessToken)
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		SetBody(`{ this is invalid }`).
		Patch(h.RestURL("/workflows/foo"))

	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest))
}

func TestWorkflowPaging(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	_, err := newWorkflowList("Bronto", 20)
	Expect(err).NotTo(HaveOccurred())

	list, _, err := client.DefaultAPI.ApiAmbientApiServerV1WorkflowsGet(ctx).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting workflow list: %v", err)
	Expect(len(list.Items)).To(Equal(20))
	Expect(list.Size).To(Equal(int32(20)))
	Expect(list.Total).To(Equal(int32(20)))
	Expect(list.Page).To(Equal(int32(1)))

	list, _, err = client.DefaultAPI.ApiAmbientApiServerV1WorkflowsGet(ctx).Page(2).Size(5).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting workflow list: %v", err)
	Expect(len(list.Items)).To(Equal(5))
	Expect(list.Size).To(Equal(int32(5)))
	Expect(list.Total).To(Equal(int32(20)))
	Expect(list.Page).To(Equal(int32(2)))
}

func TestWorkflowListSearch(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	workflows, err := newWorkflowList("bronto", 20)
	Expect(err).NotTo(HaveOccurred())

	search := fmt.Sprintf("id in ('%s')", workflows[0].ID)
	list, _, err := client.DefaultAPI.ApiAmbientApiServerV1WorkflowsGet(ctx).Search(search).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting workflow list: %v", err)
	Expect(len(list.Items)).To(Equal(1))
	Expect(list.Total).To(Equal(int32(1)))
	Expect(*list.Items[0].Id).To(Equal(workflows[0].ID))
}
