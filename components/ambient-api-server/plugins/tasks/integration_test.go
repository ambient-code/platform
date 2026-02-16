package tasks_test

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

func TestTaskGet(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	_, _, err := client.DefaultAPI.ApiAmbientApiServerV1TasksIdGet(context.Background(), "foo").Execute()
	Expect(err).To(HaveOccurred(), "Expected 401 but got nil error")

	_, resp, err := client.DefaultAPI.ApiAmbientApiServerV1TasksIdGet(ctx, "foo").Execute()
	Expect(err).To(HaveOccurred(), "Expected 404")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	taskModel, err := newTask(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	taskOutput, resp, err := client.DefaultAPI.ApiAmbientApiServerV1TasksIdGet(ctx, taskModel.ID).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	Expect(*taskOutput.Id).To(Equal(taskModel.ID), "found object does not match test object")
	Expect(*taskOutput.Kind).To(Equal("Task"))
	Expect(*taskOutput.Href).To(Equal(fmt.Sprintf("/api/ambient-api-server/v1/tasks/%s", taskModel.ID)))
	Expect(*taskOutput.CreatedAt).To(BeTemporally("~", taskModel.CreatedAt))
	Expect(*taskOutput.UpdatedAt).To(BeTemporally("~", taskModel.UpdatedAt))
}

func TestTaskPost(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	taskInput := openapi.Task{
		Name:    "test-name",
		RepoUrl: openapi.PtrString("test-repo_url"),
		Prompt:  openapi.PtrString("test-prompt"),
	}

	taskOutput, resp, err := client.DefaultAPI.ApiAmbientApiServerV1TasksPost(ctx).Task(taskInput).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))
	Expect(*taskOutput.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(*taskOutput.Kind).To(Equal("Task"))
	Expect(*taskOutput.Href).To(Equal(fmt.Sprintf("/api/ambient-api-server/v1/tasks/%s", *taskOutput.Id)))

	jwtToken := ctx.Value(openapi.ContextAccessToken)
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		SetBody(`{ this is invalid }`).
		Post(h.RestURL("/tasks"))

	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest))
}

func TestTaskPatch(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	taskModel, err := newTask(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	taskOutput, resp, err := client.DefaultAPI.ApiAmbientApiServerV1TasksIdPatch(ctx, taskModel.ID).TaskPatchRequest(openapi.TaskPatchRequest{}).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(*taskOutput.Id).To(Equal(taskModel.ID))
	Expect(*taskOutput.CreatedAt).To(BeTemporally("~", taskModel.CreatedAt))
	Expect(*taskOutput.Kind).To(Equal("Task"))
	Expect(*taskOutput.Href).To(Equal(fmt.Sprintf("/api/ambient-api-server/v1/tasks/%s", *taskOutput.Id)))

	jwtToken := ctx.Value(openapi.ContextAccessToken)
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		SetBody(`{ this is invalid }`).
		Patch(h.RestURL("/tasks/foo"))

	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest))
}

func TestTaskPaging(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	_, err := newTaskList("Bronto", 20)
	Expect(err).NotTo(HaveOccurred())

	list, _, err := client.DefaultAPI.ApiAmbientApiServerV1TasksGet(ctx).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting task list: %v", err)
	Expect(len(list.Items)).To(Equal(20))
	Expect(list.Size).To(Equal(int32(20)))
	Expect(list.Total).To(Equal(int32(20)))
	Expect(list.Page).To(Equal(int32(1)))

	list, _, err = client.DefaultAPI.ApiAmbientApiServerV1TasksGet(ctx).Page(2).Size(5).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting task list: %v", err)
	Expect(len(list.Items)).To(Equal(5))
	Expect(list.Size).To(Equal(int32(5)))
	Expect(list.Total).To(Equal(int32(20)))
	Expect(list.Page).To(Equal(int32(2)))
}

func TestTaskListSearch(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	tasks, err := newTaskList("bronto", 20)
	Expect(err).NotTo(HaveOccurred())

	search := fmt.Sprintf("id in ('%s')", tasks[0].ID)
	list, _, err := client.DefaultAPI.ApiAmbientApiServerV1TasksGet(ctx).Search(search).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting task list: %v", err)
	Expect(len(list.Items)).To(Equal(1))
	Expect(list.Total).To(Equal(int32(1)))
	Expect(*list.Items[0].Id).To(Equal(tasks[0].ID))
}
