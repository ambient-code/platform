package applications_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"gopkg.in/resty.v1"

	"github.com/ambient-code/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/ambient-code/platform/components/ambient-api-server/test"
)

func TestApplicationGet(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	_, _, err := client.DefaultAPI.ApiAmbientV1ApplicationsIdGet(context.Background(), "foo").Execute()
	Expect(err).To(HaveOccurred(), "Expected 401 but got nil error")

	_, resp, err := client.DefaultAPI.ApiAmbientV1ApplicationsIdGet(ctx, "foo").Execute()
	Expect(err).To(HaveOccurred(), "Expected 404")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	applicationModel, err := newApplication(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	applicationOutput, resp, err := client.DefaultAPI.ApiAmbientV1ApplicationsIdGet(ctx, applicationModel.ID).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	Expect(*applicationOutput.Id).To(Equal(applicationModel.ID), "found object does not match test object")
	Expect(*applicationOutput.Kind).To(Equal("Application"))
	Expect(*applicationOutput.Href).To(Equal(fmt.Sprintf("/api/ambient/v1/applications/%s", applicationModel.ID)))
	Expect(*applicationOutput.CreatedAt).To(BeTemporally("~", applicationModel.CreatedAt))
	Expect(*applicationOutput.UpdatedAt).To(BeTemporally("~", applicationModel.UpdatedAt))
}

func TestApplicationPost(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	applicationInput := openapi.Application{
		Name:                  "test-name",
		SourceRepoUrl:         "test-source_repo_url",
		SourceTargetRevision:  openapi.PtrString("test-source_target_revision"),
		SourcePath:            "test-source_path",
		DestinationAmbientUrl: openapi.PtrString("test-destination_ambient_url"),
		DestinationProject:    "test-destination_project",
		CredentialId:          openapi.PtrString("test-credential_id"),
		AutoSync:              openapi.PtrBool(true),
		AutoPrune:             openapi.PtrBool(true),
		SelfHeal:              openapi.PtrBool(true),
		SyncOptions:           openapi.PtrString("test-sync_options"),
		RetryLimit:            openapi.PtrInt32(42),
		SyncStatus:            openapi.PtrString("test-sync_status"),
		HealthStatus:          openapi.PtrString("test-health_status"),
		SyncRevision:          openapi.PtrString("test-sync_revision"),
		OperationPhase:        openapi.PtrString("test-operation_phase"),
		OperationMessage:      openapi.PtrString("test-operation_message"),
		ResourceStatus:        openapi.PtrString("test-resource_status"),
		Conditions:            openapi.PtrString("test-conditions"),
		Labels:                openapi.PtrString("test-labels"),
		Annotations:           openapi.PtrString("test-annotations"),
		LastSyncedAt:          openapi.PtrTime(time.Now()),
	}

	applicationOutput, resp, err := client.DefaultAPI.ApiAmbientV1ApplicationsPost(ctx).Application(applicationInput).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))
	Expect(*applicationOutput.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(*applicationOutput.Kind).To(Equal("Application"))
	Expect(*applicationOutput.Href).To(Equal(fmt.Sprintf("/api/ambient/v1/applications/%s", *applicationOutput.Id)))

	jwtToken := ctx.Value(openapi.ContextAccessToken)
	restyResp, _ := resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		SetBody(`{ this is invalid }`).
		Post(h.RestURL("/applications"))

	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest))
}

func TestApplicationPatch(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	applicationModel, err := newApplication(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	applicationOutput, resp, err := client.DefaultAPI.ApiAmbientV1ApplicationsIdPatch(ctx, applicationModel.ID).ApplicationPatchRequest(openapi.ApplicationPatchRequest{}).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(*applicationOutput.Id).To(Equal(applicationModel.ID))
	Expect(*applicationOutput.CreatedAt).To(BeTemporally("~", applicationModel.CreatedAt))
	Expect(*applicationOutput.Kind).To(Equal("Application"))
	Expect(*applicationOutput.Href).To(Equal(fmt.Sprintf("/api/ambient/v1/applications/%s", *applicationOutput.Id)))

	jwtToken := ctx.Value(openapi.ContextAccessToken)
	restyResp, _ := resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		SetBody(`{ this is invalid }`).
		Patch(h.RestURL("/applications/foo"))

	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest))
}

func TestApplicationPaging(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	_, err := newApplicationList("Bronto", 20)
	Expect(err).NotTo(HaveOccurred())

	list, _, err := client.DefaultAPI.ApiAmbientV1ApplicationsGet(ctx).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting application list: %v", err)
	Expect(len(list.Items)).To(Equal(20))
	Expect(list.Size).To(Equal(int32(20)))
	Expect(list.Total).To(Equal(int32(20)))
	Expect(list.Page).To(Equal(int32(1)))

	list, _, err = client.DefaultAPI.ApiAmbientV1ApplicationsGet(ctx).Page(2).Size(5).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting application list: %v", err)
	Expect(len(list.Items)).To(Equal(5))
	Expect(list.Size).To(Equal(int32(5)))
	Expect(list.Total).To(Equal(int32(20)))
	Expect(list.Page).To(Equal(int32(2)))
}

func TestApplicationListSearch(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	applications, err := newApplicationList("bronto", 20)
	Expect(err).NotTo(HaveOccurred())

	search := fmt.Sprintf("id in ('%s')", applications[0].ID)
	list, _, err := client.DefaultAPI.ApiAmbientV1ApplicationsGet(ctx).Search(search).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting application list: %v", err)
	Expect(len(list.Items)).To(Equal(1))
	Expect(list.Total).To(Equal(int32(1)))
	Expect(*list.Items[0].Id).To(Equal(applications[0].ID))
}
