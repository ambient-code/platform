package repositoryRefs_test

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

func TestRepositoryRefGet(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	_, _, err := client.DefaultAPI.ApiAmbientApiServerV1RepositoryRefsIdGet(context.Background(), "foo").Execute()
	Expect(err).To(HaveOccurred(), "Expected 401 but got nil error")

	_, resp, err := client.DefaultAPI.ApiAmbientApiServerV1RepositoryRefsIdGet(ctx, "foo").Execute()
	Expect(err).To(HaveOccurred(), "Expected 404")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	repositoryRefModel, err := newRepositoryRef(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	repositoryRefOutput, resp, err := client.DefaultAPI.ApiAmbientApiServerV1RepositoryRefsIdGet(ctx, repositoryRefModel.ID).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	Expect(*repositoryRefOutput.Id).To(Equal(repositoryRefModel.ID), "found object does not match test object")
	Expect(*repositoryRefOutput.Kind).To(Equal("RepositoryRef"))
	Expect(*repositoryRefOutput.Href).To(Equal(fmt.Sprintf("/api/ambient-api-server/v1/repository_refs/%s", repositoryRefModel.ID)))
	Expect(*repositoryRefOutput.CreatedAt).To(BeTemporally("~", repositoryRefModel.CreatedAt))
	Expect(*repositoryRefOutput.UpdatedAt).To(BeTemporally("~", repositoryRefModel.UpdatedAt))
}

func TestRepositoryRefPost(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	repositoryRefInput := openapi.RepositoryRef{
		Name:      "test-name",
		Url:       "test-url",
		Branch:    openapi.PtrString("test-branch"),
		Provider:  openapi.PtrString("test-provider"),
		Owner:     openapi.PtrString("test-owner"),
		RepoName:  openapi.PtrString("test-repo_name"),
		ProjectId: openapi.PtrString("test-project_id"),
	}

	repositoryRefOutput, resp, err := client.DefaultAPI.ApiAmbientApiServerV1RepositoryRefsPost(ctx).RepositoryRef(repositoryRefInput).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))
	Expect(*repositoryRefOutput.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(*repositoryRefOutput.Kind).To(Equal("RepositoryRef"))
	Expect(*repositoryRefOutput.Href).To(Equal(fmt.Sprintf("/api/ambient-api-server/v1/repository_refs/%s", *repositoryRefOutput.Id)))

	jwtToken := ctx.Value(openapi.ContextAccessToken)
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		SetBody(`{ this is invalid }`).
		Post(h.RestURL("/repository_refs"))

	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest))
}

func TestRepositoryRefPatch(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	repositoryRefModel, err := newRepositoryRef(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	repositoryRefOutput, resp, err := client.DefaultAPI.ApiAmbientApiServerV1RepositoryRefsIdPatch(ctx, repositoryRefModel.ID).RepositoryRefPatchRequest(openapi.RepositoryRefPatchRequest{}).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(*repositoryRefOutput.Id).To(Equal(repositoryRefModel.ID))
	Expect(*repositoryRefOutput.CreatedAt).To(BeTemporally("~", repositoryRefModel.CreatedAt))
	Expect(*repositoryRefOutput.Kind).To(Equal("RepositoryRef"))
	Expect(*repositoryRefOutput.Href).To(Equal(fmt.Sprintf("/api/ambient-api-server/v1/repository_refs/%s", *repositoryRefOutput.Id)))

	jwtToken := ctx.Value(openapi.ContextAccessToken)
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		SetBody(`{ this is invalid }`).
		Patch(h.RestURL("/repository_refs/foo"))

	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest))
}

func TestRepositoryRefPaging(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	_, err := newRepositoryRefList("Bronto", 20)
	Expect(err).NotTo(HaveOccurred())

	list, _, err := client.DefaultAPI.ApiAmbientApiServerV1RepositoryRefsGet(ctx).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting repositoryRef list: %v", err)
	Expect(len(list.Items)).To(Equal(20))
	Expect(list.Size).To(Equal(int32(20)))
	Expect(list.Total).To(Equal(int32(20)))
	Expect(list.Page).To(Equal(int32(1)))

	list, _, err = client.DefaultAPI.ApiAmbientApiServerV1RepositoryRefsGet(ctx).Page(2).Size(5).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting repositoryRef list: %v", err)
	Expect(len(list.Items)).To(Equal(5))
	Expect(list.Size).To(Equal(int32(5)))
	Expect(list.Total).To(Equal(int32(20)))
	Expect(list.Page).To(Equal(int32(2)))
}

func TestRepositoryRefListSearch(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	repositoryRefs, err := newRepositoryRefList("bronto", 20)
	Expect(err).NotTo(HaveOccurred())

	search := fmt.Sprintf("id in ('%s')", repositoryRefs[0].ID)
	list, _, err := client.DefaultAPI.ApiAmbientApiServerV1RepositoryRefsGet(ctx).Search(search).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting repositoryRef list: %v", err)
	Expect(len(list.Items)).To(Equal(1))
	Expect(list.Total).To(Equal(int32(1)))
	Expect(*list.Items[0].Id).To(Equal(repositoryRefs[0].ID))
}
