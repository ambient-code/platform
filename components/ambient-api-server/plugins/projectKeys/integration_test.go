package projectKeys_test

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

func TestProjectKeyGet(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	_, _, err := client.DefaultAPI.ApiAmbientApiServerV1ProjectKeysIdGet(context.Background(), "foo").Execute()
	Expect(err).To(HaveOccurred(), "Expected 401 but got nil error")

	_, resp, err := client.DefaultAPI.ApiAmbientApiServerV1ProjectKeysIdGet(ctx, "foo").Execute()
	Expect(err).To(HaveOccurred(), "Expected 404")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	projectKeyModel, err := newProjectKey("test-get-key")
	Expect(err).NotTo(HaveOccurred())

	projectKeyOutput, resp, err := client.DefaultAPI.ApiAmbientApiServerV1ProjectKeysIdGet(ctx, projectKeyModel.ID).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	Expect(*projectKeyOutput.Id).To(Equal(projectKeyModel.ID), "found object does not match test object")
	Expect(*projectKeyOutput.Kind).To(Equal("ProjectKey"))
	Expect(*projectKeyOutput.Href).To(Equal(fmt.Sprintf("/api/ambient-api-server/v1/project_keys/%s", projectKeyModel.ID)))
	Expect(*projectKeyOutput.CreatedAt).To(BeTemporally("~", projectKeyModel.CreatedAt))
	Expect(*projectKeyOutput.UpdatedAt).To(BeTemporally("~", projectKeyModel.UpdatedAt))

	Expect(projectKeyOutput.PlaintextKey).To(BeNil(), "plaintext_key must NOT be returned on GET")
	Expect(*projectKeyOutput.KeyPrefix).NotTo(BeEmpty(), "key_prefix should be present on GET")
}

func TestProjectKeyPost(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	projectKeyInput := openapi.ProjectKey{
		Name:      "test-post-key",
		ProjectId: openapi.PtrString("test-project"),
	}

	projectKeyOutput, resp, err := client.DefaultAPI.ApiAmbientApiServerV1ProjectKeysPost(ctx).ProjectKey(projectKeyInput).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error posting object: %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))
	Expect(*projectKeyOutput.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(*projectKeyOutput.Kind).To(Equal("ProjectKey"))
	Expect(*projectKeyOutput.Href).To(Equal(fmt.Sprintf("/api/ambient-api-server/v1/project_keys/%s", *projectKeyOutput.Id)))

	Expect(projectKeyOutput.PlaintextKey).NotTo(BeNil(), "plaintext_key MUST be returned on create")
	Expect(*projectKeyOutput.PlaintextKey).To(HavePrefix("ak_"), "plaintext_key must start with ak_ prefix")
	Expect(len(*projectKeyOutput.PlaintextKey)).To(BeNumerically(">", 20), "plaintext_key must be substantial length")
	Expect(*projectKeyOutput.KeyPrefix).To(Equal((*projectKeyOutput.PlaintextKey)[:8]), "key_prefix must match first 8 chars of plaintext_key")

	jwtToken := ctx.Value(openapi.ContextAccessToken)
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		SetBody(`{ this is invalid }`).
		Post(h.RestURL("/project_keys"))

	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest))
}

func TestProjectKeyPlaintextKeyNotOnGet(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	projectKeyInput := openapi.ProjectKey{
		Name: "test-plaintext-security",
	}

	createOutput, _, err := client.DefaultAPI.ApiAmbientApiServerV1ProjectKeysPost(ctx).ProjectKey(projectKeyInput).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(createOutput.PlaintextKey).NotTo(BeNil(), "plaintext_key must be in create response")

	getOutput, _, err := client.DefaultAPI.ApiAmbientApiServerV1ProjectKeysIdGet(ctx, *createOutput.Id).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(getOutput.PlaintextKey).To(BeNil(), "plaintext_key must NOT be in GET response")
}

func TestProjectKeyDelete(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	projectKeyModel, err := newProjectKey("test-delete-key")
	Expect(err).NotTo(HaveOccurred())

	resp, err := client.DefaultAPI.ApiAmbientApiServerV1ProjectKeysIdDelete(ctx, projectKeyModel.ID).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusNoContent))

	_, resp, err = client.DefaultAPI.ApiAmbientApiServerV1ProjectKeysIdGet(ctx, projectKeyModel.ID).Execute()
	Expect(err).To(HaveOccurred(), "Expected 404 after delete")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
}

func TestProjectKeyPaging(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	_, err := newProjectKeyList("PagingKey", 20)
	Expect(err).NotTo(HaveOccurred())

	list, _, err := client.DefaultAPI.ApiAmbientApiServerV1ProjectKeysGet(ctx).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting project key list: %v", err)
	Expect(len(list.Items)).To(Equal(20))
	Expect(list.Size).To(Equal(int32(20)))
	Expect(list.Total).To(Equal(int32(20)))
	Expect(list.Page).To(Equal(int32(1)))

	list, _, err = client.DefaultAPI.ApiAmbientApiServerV1ProjectKeysGet(ctx).Page(2).Size(5).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting project key list: %v", err)
	Expect(len(list.Items)).To(Equal(5))
	Expect(list.Size).To(Equal(int32(5)))
	Expect(list.Total).To(Equal(int32(20)))
	Expect(list.Page).To(Equal(int32(2)))
}

func TestProjectKeyListSearch(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	projectKeys, err := newProjectKeyList("SearchKey", 20)
	Expect(err).NotTo(HaveOccurred())

	search := fmt.Sprintf("id in ('%s')", projectKeys[0].ID)
	list, _, err := client.DefaultAPI.ApiAmbientApiServerV1ProjectKeysGet(ctx).Search(search).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting project key list: %v", err)
	Expect(len(list.Items)).To(Equal(1))
	Expect(list.Total).To(Equal(int32(1)))
	Expect(*list.Items[0].Id).To(Equal(projectKeys[0].ID))
}
