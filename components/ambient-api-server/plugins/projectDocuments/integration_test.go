package projectDocuments_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	. "github.com/onsi/gomega"
	"gopkg.in/resty.v1"

	"github.com/ambient-code/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/ambient-code/platform/components/ambient-api-server/test"
)

func TestProjectDocumentGet(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	_, _, err := client.DefaultAPI.ApiAmbientV1ProjectDocumentsIdGet(context.Background(), "foo").Execute()
	Expect(err).To(HaveOccurred(), "Expected 401 but got nil error")

	_, resp, err := client.DefaultAPI.ApiAmbientV1ProjectDocumentsIdGet(ctx, "foo").Execute()
	Expect(err).To(HaveOccurred(), "Expected 404")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	projectDocumentModel, err := newProjectDocument(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	projectDocumentOutput, resp, err := client.DefaultAPI.ApiAmbientV1ProjectDocumentsIdGet(ctx, projectDocumentModel.ID).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	Expect(*projectDocumentOutput.Id).To(Equal(projectDocumentModel.ID), "found object does not match test object")
	Expect(*projectDocumentOutput.Kind).To(Equal("ProjectDocument"))
	Expect(*projectDocumentOutput.Href).To(Equal(fmt.Sprintf("/api/ambient/v1/project_documents/%s", projectDocumentModel.ID)))
	Expect(*projectDocumentOutput.CreatedAt).To(BeTemporally("~", projectDocumentModel.CreatedAt))
	Expect(*projectDocumentOutput.UpdatedAt).To(BeTemporally("~", projectDocumentModel.UpdatedAt))
}

func TestProjectDocumentPost(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	projectDocumentInput := openapi.ProjectDocument{
		ProjectId: "test-project_id",
		Slug:      "test-slug",
		Title:     openapi.PtrString("test-title"),
		Content:   openapi.PtrString("test-content"),
	}

	projectDocumentOutput, resp, err := client.DefaultAPI.ApiAmbientV1ProjectDocumentsPost(ctx).ProjectDocument(projectDocumentInput).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))
	Expect(*projectDocumentOutput.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(*projectDocumentOutput.Kind).To(Equal("ProjectDocument"))
	Expect(*projectDocumentOutput.Href).To(Equal(fmt.Sprintf("/api/ambient/v1/project_documents/%s", *projectDocumentOutput.Id)))

	jwtToken := ctx.Value(openapi.ContextAccessToken)
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		SetBody(`{ this is invalid }`).
		Post(h.RestURL("/project_documents"))

	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest))
}

func TestProjectDocumentPatch(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	projectDocumentModel, err := newProjectDocument(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	projectDocumentOutput, resp, err := client.DefaultAPI.ApiAmbientV1ProjectDocumentsIdPatch(ctx, projectDocumentModel.ID).ProjectDocumentPatchRequest(openapi.ProjectDocumentPatchRequest{}).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(*projectDocumentOutput.Id).To(Equal(projectDocumentModel.ID))
	Expect(*projectDocumentOutput.CreatedAt).To(BeTemporally("~", projectDocumentModel.CreatedAt))
	Expect(*projectDocumentOutput.Kind).To(Equal("ProjectDocument"))
	Expect(*projectDocumentOutput.Href).To(Equal(fmt.Sprintf("/api/ambient/v1/project_documents/%s", *projectDocumentOutput.Id)))

	jwtToken := ctx.Value(openapi.ContextAccessToken)
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		SetBody(`{ this is invalid }`).
		Patch(h.RestURL("/project_documents/foo"))

	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest))
}

func TestProjectDocumentPaging(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	_, err := newProjectDocumentList("Bronto", 20)
	Expect(err).NotTo(HaveOccurred())

	list, _, err := client.DefaultAPI.ApiAmbientV1ProjectDocumentsGet(ctx).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting projectDocument list: %v", err)
	Expect(len(list.Items)).To(Equal(20))
	Expect(list.Size).To(Equal(int32(20)))
	Expect(list.Total).To(Equal(int32(20)))
	Expect(list.Page).To(Equal(int32(1)))

	list, _, err = client.DefaultAPI.ApiAmbientV1ProjectDocumentsGet(ctx).Page(2).Size(5).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting projectDocument list: %v", err)
	Expect(len(list.Items)).To(Equal(5))
	Expect(list.Size).To(Equal(int32(5)))
	Expect(list.Total).To(Equal(int32(20)))
	Expect(list.Page).To(Equal(int32(2)))
}

func TestProjectDocumentListSearch(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	projectDocuments, err := newProjectDocumentList("bronto", 20)
	Expect(err).NotTo(HaveOccurred())

	search := fmt.Sprintf("id in ('%s')", projectDocuments[0].ID)
	list, _, err := client.DefaultAPI.ApiAmbientV1ProjectDocumentsGet(ctx).Search(search).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting projectDocument list: %v", err)
	Expect(len(list.Items)).To(Equal(1))
	Expect(list.Total).To(Equal(int32(1)))
	Expect(*list.Items[0].Id).To(Equal(projectDocuments[0].ID))
}
