package projects_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	. "github.com/onsi/gomega"
	"gopkg.in/resty.v1"

	"github.com/ambient/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/ambient/platform/components/ambient-api-server/plugins/projects"
	"github.com/ambient/platform/components/ambient-api-server/test"
	"github.com/openshift-online/rh-trex-ai/pkg/environments"
)

func TestProjectGet(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	_, _, err := client.DefaultAPI.ApiAmbientV1ProjectsIdGet(context.Background(), "foo").Execute()
	Expect(err).To(HaveOccurred(), "Expected 401 but got nil error")

	_, resp, err := client.DefaultAPI.ApiAmbientV1ProjectsIdGet(ctx, "foo").Execute()
	Expect(err).To(HaveOccurred(), "Expected 404")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	projectModel, err := newProject("get")
	Expect(err).NotTo(HaveOccurred())

	projectOutput, resp, err := client.DefaultAPI.ApiAmbientV1ProjectsIdGet(ctx, projectModel.ID).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	Expect(*projectOutput.Id).To(Equal(projectModel.ID), "found object does not match test object")
	Expect(*projectOutput.Kind).To(Equal("Project"))
	Expect(*projectOutput.Href).To(Equal(fmt.Sprintf("/api/ambient/v1/projects/%s", projectModel.ID)))
	Expect(*projectOutput.CreatedAt).To(BeTemporally("~", projectModel.CreatedAt))
	Expect(*projectOutput.UpdatedAt).To(BeTemporally("~", projectModel.UpdatedAt))
}

func TestProjectPost(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	projectInput := openapi.Project{
		Name:        "test-project",
		DisplayName: openapi.PtrString("Test Project"),
		Description: openapi.PtrString("test-description"),
		Status:      openapi.PtrString("active"),
	}

	projectOutput, resp, err := client.DefaultAPI.ApiAmbientV1ProjectsPost(ctx).Project(projectInput).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))
	Expect(*projectOutput.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(*projectOutput.Kind).To(Equal("Project"))
	Expect(*projectOutput.Href).To(Equal(fmt.Sprintf("/api/ambient/v1/projects/%s", *projectOutput.Id)))

	jwtToken := ctx.Value(openapi.ContextAccessToken)
	restyResp, _ := resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		SetBody(`{ this is invalid }`).
		Post(h.RestURL("/projects"))

	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest))
}

func TestProjectPatch(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	projectModel, err := newProject("patch")
	Expect(err).NotTo(HaveOccurred())

	projectOutput, resp, err := client.DefaultAPI.ApiAmbientV1ProjectsIdPatch(ctx, projectModel.ID).ProjectPatchRequest(openapi.ProjectPatchRequest{}).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(*projectOutput.Id).To(Equal(projectModel.ID))
	Expect(*projectOutput.CreatedAt).To(BeTemporally("~", projectModel.CreatedAt))
	Expect(*projectOutput.Kind).To(Equal("Project"))
	Expect(*projectOutput.Href).To(Equal(fmt.Sprintf("/api/ambient/v1/projects/%s", *projectOutput.Id)))

	jwtToken := ctx.Value(openapi.ContextAccessToken)
	restyResp, _ := resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		SetBody(`{ this is invalid }`).
		Patch(h.RestURL("/projects/foo"))

	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest))
}

func TestProjectPaging(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	_, err := newProjectList("paging", 20)
	Expect(err).NotTo(HaveOccurred())

	list, _, err := client.DefaultAPI.ApiAmbientV1ProjectsGet(ctx).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting project list: %v", err)
	Expect(len(list.Items)).To(Equal(20))
	Expect(list.Size).To(Equal(int32(20)))
	Expect(list.Total).To(Equal(int32(20)))
	Expect(list.Page).To(Equal(int32(1)))

	list, _, err = client.DefaultAPI.ApiAmbientV1ProjectsGet(ctx).Page(2).Size(5).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting project list: %v", err)
	Expect(len(list.Items)).To(Equal(5))
	Expect(list.Size).To(Equal(int32(5)))
	Expect(list.Total).To(Equal(int32(20)))
	Expect(list.Page).To(Equal(int32(2)))
}

func TestProjectAll(t *testing.T) {
	h, _ := test.RegisterIntegration(t)
	_ = h

	created, err := newProjectList("all", 5)
	Expect(err).NotTo(HaveOccurred())

	projectService := projects.Service(&environments.Environment().Services)
	all, svcErr := projectService.All(context.Background())
	Expect(svcErr).NotTo(HaveOccurred(), "Error calling All(): %v", svcErr)
	Expect(len(all)).To(Equal(5))

	returnedIDs := map[string]bool{}
	for _, p := range all {
		returnedIDs[p.ID] = true
	}
	for _, p := range created {
		Expect(returnedIDs).To(HaveKey(p.ID), "All() should return project %s", p.ID)
	}
}

func TestProjectAllEmpty(t *testing.T) {
	h, _ := test.RegisterIntegration(t)
	_ = h

	projectService := projects.Service(&environments.Environment().Services)
	all, svcErr := projectService.All(context.Background())
	Expect(svcErr).NotTo(HaveOccurred(), "Error calling All() on empty table: %v", svcErr)
	Expect(len(all)).To(Equal(0))
}

func TestProjectListSearch(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	projects, err := newProjectList("search", 20)
	Expect(err).NotTo(HaveOccurred())

	search := fmt.Sprintf("id in ('%s')", projects[0].ID)
	list, _, err := client.DefaultAPI.ApiAmbientV1ProjectsGet(ctx).Search(search).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting project list: %v", err)
	Expect(len(list.Items)).To(Equal(1))
	Expect(list.Total).To(Equal(int32(1)))
	Expect(*list.Items[0].Id).To(Equal(projects[0].ID))
}
