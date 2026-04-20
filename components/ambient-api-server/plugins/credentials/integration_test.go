package credentials_test

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

const testProjectID = "test-project"

func TestCredentialGet(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	_, _, err := client.DefaultAPI.ApiAmbientV1ProjectsIdCredentialsCredIdGet(context.Background(), testProjectID, "foo").Execute()
	Expect(err).To(HaveOccurred(), "Expected 401 but got nil error")

	_, resp, err := client.DefaultAPI.ApiAmbientV1ProjectsIdCredentialsCredIdGet(ctx, testProjectID, "foo").Execute()
	Expect(err).To(HaveOccurred(), "Expected 404")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	credentialModel, err := newCredential(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	credentialOutput, resp, err := client.DefaultAPI.ApiAmbientV1ProjectsIdCredentialsCredIdGet(ctx, testProjectID, credentialModel.ID).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	Expect(*credentialOutput.Id).To(Equal(credentialModel.ID), "found object does not match test object")
	Expect(*credentialOutput.Kind).To(Equal("Credential"))
	Expect(*credentialOutput.Href).To(Equal(fmt.Sprintf("/api/ambient/v1/credentials/%s", credentialModel.ID)))
	Expect(*credentialOutput.CreatedAt).To(BeTemporally("~", credentialModel.CreatedAt))
	Expect(*credentialOutput.UpdatedAt).To(BeTemporally("~", credentialModel.UpdatedAt))
	Expect(credentialOutput.Token).To(BeNil(), "GET must never return the token value")
}

func TestCredentialPost(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	credentialInput := openapi.Credential{
		ProjectId:   testProjectID,
		Name:        "test-name",
		Description: openapi.PtrString("test-description"),
		Provider:    "test-provider",
		Token:       openapi.PtrString("test-token"),
		Url:         openapi.PtrString("test-url"),
		Email:       openapi.PtrString("test-email"),
		Labels:      openapi.PtrString("test-labels"),
		Annotations: openapi.PtrString("test-annotations"),
	}

	credentialOutput, resp, err := client.DefaultAPI.ApiAmbientV1ProjectsIdCredentialsPost(ctx, testProjectID).Credential(credentialInput).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))
	Expect(*credentialOutput.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(*credentialOutput.Kind).To(Equal("Credential"))
	Expect(*credentialOutput.Href).To(Equal(fmt.Sprintf("/api/ambient/v1/credentials/%s", *credentialOutput.Id)))
	Expect(credentialOutput.Token).To(BeNil(), "POST response must never return the token value")

	jwtToken := ctx.Value(openapi.ContextAccessToken)
	restyResp, restyErr := resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		SetBody(`{ this is invalid }`).
		Post(h.RestURL(fmt.Sprintf("/projects/%s/credentials", testProjectID)))

	Expect(restyErr).NotTo(HaveOccurred())
	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest))
}

func TestCredentialPatch(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	credentialModel, err := newCredential(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	credentialOutput, resp, err := client.DefaultAPI.ApiAmbientV1ProjectsIdCredentialsCredIdPatch(ctx, testProjectID, credentialModel.ID).CredentialPatchRequest(openapi.CredentialPatchRequest{}).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(*credentialOutput.Id).To(Equal(credentialModel.ID))
	Expect(*credentialOutput.CreatedAt).To(BeTemporally("~", credentialModel.CreatedAt))
	Expect(*credentialOutput.Kind).To(Equal("Credential"))
	Expect(*credentialOutput.Href).To(Equal(fmt.Sprintf("/api/ambient/v1/credentials/%s", *credentialOutput.Id)))

	jwtToken := ctx.Value(openapi.ContextAccessToken)
	restyResp, restyErr := resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		SetBody(`{ this is invalid }`).
		Patch(h.RestURL(fmt.Sprintf("/projects/%s/credentials/foo", testProjectID)))

	Expect(restyErr).NotTo(HaveOccurred())
	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest))
}

func TestCredentialPaging(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	_, err := newCredentialList("Bronto", 20)
	Expect(err).NotTo(HaveOccurred())

	list, _, err := client.DefaultAPI.ApiAmbientV1ProjectsIdCredentialsGet(ctx, testProjectID).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting credential list: %v", err)
	Expect(len(list.Items)).To(Equal(20))
	Expect(list.Size).To(Equal(int32(20)))
	Expect(list.Total).To(Equal(int32(20)))
	Expect(list.Page).To(Equal(int32(1)))

	list, _, err = client.DefaultAPI.ApiAmbientV1ProjectsIdCredentialsGet(ctx, testProjectID).Page(2).Size(5).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting credential list: %v", err)
	Expect(len(list.Items)).To(Equal(5))
	Expect(list.Size).To(Equal(int32(5)))
	Expect(list.Total).To(Equal(int32(20)))
	Expect(list.Page).To(Equal(int32(2)))
}

func TestCredentialListSearch(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	credentials, err := newCredentialList("bronto", 20)
	Expect(err).NotTo(HaveOccurred())

	search := fmt.Sprintf("id in ('%s')", credentials[0].ID)
	list, _, err := client.DefaultAPI.ApiAmbientV1ProjectsIdCredentialsGet(ctx, testProjectID).Search(search).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting credential list: %v", err)
	Expect(len(list.Items)).To(Equal(1))
	Expect(list.Total).To(Equal(int32(1)))
	Expect(*list.Items[0].Id).To(Equal(credentials[0].ID))
}

func TestCredentialListTokenOmitted(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	created, err := newCredential(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	list, _, err := client.DefaultAPI.ApiAmbientV1ProjectsIdCredentialsGet(ctx, testProjectID).Execute()
	Expect(err).NotTo(HaveOccurred())

	var found *openapi.Credential
	for i := range list.Items {
		if *list.Items[i].Id == created.ID {
			found = &list.Items[i]
			break
		}
	}
	Expect(found).NotTo(BeNil(), "created credential must appear in list")
	Expect(found.Token).To(BeNil(), "LIST must never return the token value")
}

func TestCredentialToken(t *testing.T) {
	h, _ := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	created, err := newCredential(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	jwtToken := ctx.Value(openapi.ContextAccessToken)

	restyResp, restyErr := resty.R().
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		Get(h.RestURL(fmt.Sprintf("/projects/%s/credentials/%s/token", testProjectID, created.ID)))
	Expect(restyErr).NotTo(HaveOccurred())
	Expect(restyResp.StatusCode()).To(Equal(http.StatusOK))
	Expect(restyResp.String()).To(ContainSubstring(`"token"`))
	Expect(restyResp.String()).To(ContainSubstring(`"provider"`))
	Expect(restyResp.String()).To(ContainSubstring(`"credential_id"`))

	restyResp, restyErr = resty.R().
		Get(h.RestURL(fmt.Sprintf("/projects/%s/credentials/%s/token", testProjectID, created.ID)))
	Expect(restyErr).NotTo(HaveOccurred())
	Expect(restyResp.StatusCode()).To(Equal(http.StatusUnauthorized), "unauthenticated request to /token must be rejected")
}

func TestCredentialDelete(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	created, err := newCredential(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	_, resp, err := client.DefaultAPI.ApiAmbientV1ProjectsIdCredentialsCredIdGet(ctx, testProjectID, created.ID).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	resp, err = client.DefaultAPI.ApiAmbientV1ProjectsIdCredentialsCredIdDelete(ctx, testProjectID, created.ID).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusNoContent))

	_, resp, err = client.DefaultAPI.ApiAmbientV1ProjectsIdCredentialsCredIdGet(ctx, testProjectID, created.ID).Execute()
	Expect(err).To(HaveOccurred(), "Expected 404 after delete")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	resp, err = client.DefaultAPI.ApiAmbientV1ProjectsIdCredentialsCredIdDelete(context.Background(), testProjectID, created.ID).Execute()
	Expect(err).To(HaveOccurred(), "Expected 401 for unauthenticated delete")
	_ = resp
}
