package workflowTasks_test

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

func TestWorkflowTaskGet(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	_, _, err := client.DefaultAPI.ApiAmbientApiServerV1WorkflowTasksIdGet(context.Background(), "foo").Execute()
	Expect(err).To(HaveOccurred(), "Expected 401 but got nil error")

	_, resp, err := client.DefaultAPI.ApiAmbientApiServerV1WorkflowTasksIdGet(ctx, "foo").Execute()
	Expect(err).To(HaveOccurred(), "Expected 404")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	workflowTaskModel, err := newWorkflowTask(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	workflowTaskOutput, resp, err := client.DefaultAPI.ApiAmbientApiServerV1WorkflowTasksIdGet(ctx, workflowTaskModel.ID).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	Expect(*workflowTaskOutput.Id).To(Equal(workflowTaskModel.ID), "found object does not match test object")
	Expect(*workflowTaskOutput.Kind).To(Equal("WorkflowTask"))
	Expect(*workflowTaskOutput.Href).To(Equal(fmt.Sprintf("/api/ambient-api-server/v1/workflow_tasks/%s", workflowTaskModel.ID)))
	Expect(*workflowTaskOutput.CreatedAt).To(BeTemporally("~", workflowTaskModel.CreatedAt))
	Expect(*workflowTaskOutput.UpdatedAt).To(BeTemporally("~", workflowTaskModel.UpdatedAt))
}

func TestWorkflowTaskPost(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	wf, err := newParentWorkflow()
	Expect(err).NotTo(HaveOccurred(), "Error creating parent workflow")
	tk, err := newParentTask()
	Expect(err).NotTo(HaveOccurred(), "Error creating parent task")

	workflowTaskInput := openapi.WorkflowTask{
		WorkflowId: wf.ID,
		TaskId:     tk.ID,
		Position:   42,
	}

	workflowTaskOutput, resp, err := client.DefaultAPI.ApiAmbientApiServerV1WorkflowTasksPost(ctx).WorkflowTask(workflowTaskInput).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))
	Expect(*workflowTaskOutput.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(*workflowTaskOutput.Kind).To(Equal("WorkflowTask"))
	Expect(*workflowTaskOutput.Href).To(Equal(fmt.Sprintf("/api/ambient-api-server/v1/workflow_tasks/%s", *workflowTaskOutput.Id)))

	jwtToken := ctx.Value(openapi.ContextAccessToken)
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		SetBody(`{ this is invalid }`).
		Post(h.RestURL("/workflow_tasks"))

	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest))
}

func TestWorkflowTaskPatch(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	workflowTaskModel, err := newWorkflowTask(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	workflowTaskOutput, resp, err := client.DefaultAPI.ApiAmbientApiServerV1WorkflowTasksIdPatch(ctx, workflowTaskModel.ID).WorkflowTaskPatchRequest(openapi.WorkflowTaskPatchRequest{}).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(*workflowTaskOutput.Id).To(Equal(workflowTaskModel.ID))
	Expect(*workflowTaskOutput.CreatedAt).To(BeTemporally("~", workflowTaskModel.CreatedAt))
	Expect(*workflowTaskOutput.Kind).To(Equal("WorkflowTask"))
	Expect(*workflowTaskOutput.Href).To(Equal(fmt.Sprintf("/api/ambient-api-server/v1/workflow_tasks/%s", *workflowTaskOutput.Id)))

	jwtToken := ctx.Value(openapi.ContextAccessToken)
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		SetBody(`{ this is invalid }`).
		Patch(h.RestURL("/workflow_tasks/foo"))

	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest))
}

func TestWorkflowTaskPaging(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	_, err := newWorkflowTaskList("Bronto", 20)
	Expect(err).NotTo(HaveOccurred())

	list, _, err := client.DefaultAPI.ApiAmbientApiServerV1WorkflowTasksGet(ctx).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting workflowTask list: %v", err)
	Expect(len(list.Items)).To(Equal(20))
	Expect(list.Size).To(Equal(int32(20)))
	Expect(list.Total).To(Equal(int32(20)))
	Expect(list.Page).To(Equal(int32(1)))

	list, _, err = client.DefaultAPI.ApiAmbientApiServerV1WorkflowTasksGet(ctx).Page(2).Size(5).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting workflowTask list: %v", err)
	Expect(len(list.Items)).To(Equal(5))
	Expect(list.Size).To(Equal(int32(5)))
	Expect(list.Total).To(Equal(int32(20)))
	Expect(list.Page).To(Equal(int32(2)))
}

func TestWorkflowTaskListSearch(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	workflowTasks, err := newWorkflowTaskList("bronto", 20)
	Expect(err).NotTo(HaveOccurred())

	search := fmt.Sprintf("id in ('%s')", workflowTasks[0].ID)
	list, _, err := client.DefaultAPI.ApiAmbientApiServerV1WorkflowTasksGet(ctx).Search(search).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting workflowTask list: %v", err)
	Expect(len(list.Items)).To(Equal(1))
	Expect(list.Total).To(Equal(int32(1)))
	Expect(*list.Items[0].Id).To(Equal(workflowTasks[0].ID))
}
