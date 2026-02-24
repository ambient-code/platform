package projectSettings_test

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

func TestProjectSettingsGet(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	_, _, err := client.DefaultAPI.ApiAmbientApiServerV1ProjectSettingsIdGet(context.Background(), "foo").Execute()
	Expect(err).To(HaveOccurred(), "Expected 401 but got nil error")

	_, resp, err := client.DefaultAPI.ApiAmbientApiServerV1ProjectSettingsIdGet(ctx, "foo").Execute()
	Expect(err).To(HaveOccurred(), "Expected 404")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	psModel, err := newProjectSettings(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	psOutput, resp, err := client.DefaultAPI.ApiAmbientApiServerV1ProjectSettingsIdGet(ctx, psModel.ID).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	Expect(*psOutput.Id).To(Equal(psModel.ID), "found object does not match test object")
	Expect(*psOutput.Kind).To(Equal("ProjectSettings"))
	Expect(*psOutput.Href).To(Equal(fmt.Sprintf("/api/ambient-api-server/v1/project_settings/%s", psModel.ID)))
	Expect(*psOutput.CreatedAt).To(BeTemporally("~", psModel.CreatedAt))
	Expect(*psOutput.UpdatedAt).To(BeTemporally("~", psModel.UpdatedAt))
}

func TestProjectSettingsPost(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	project, err := newParentProject()
	Expect(err).NotTo(HaveOccurred())

	psInput := openapi.ProjectSettings{
		ProjectId:    project.ID,
		GroupAccess:  openapi.PtrString(`[{"group_name":"devs","role":"edit"}]`),
		Repositories: openapi.PtrString(`[]`),
	}

	psOutput, resp, err := client.DefaultAPI.ApiAmbientApiServerV1ProjectSettingsPost(ctx).ProjectSettings(psInput).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))
	Expect(*psOutput.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(*psOutput.Kind).To(Equal("ProjectSettings"))
	Expect(*psOutput.Href).To(Equal(fmt.Sprintf("/api/ambient-api-server/v1/project_settings/%s", *psOutput.Id)))

	jwtToken := ctx.Value(openapi.ContextAccessToken)
	restyResp, _ := resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		SetBody(`{ this is invalid }`).
		Post(h.RestURL("/project_settings"))

	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest))
}

func TestProjectSettingsPatch(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	psModel, err := newProjectSettings(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	psOutput, resp, err := client.DefaultAPI.ApiAmbientApiServerV1ProjectSettingsIdPatch(ctx, psModel.ID).ProjectSettingsPatchRequest(openapi.ProjectSettingsPatchRequest{}).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(*psOutput.Id).To(Equal(psModel.ID))
	Expect(*psOutput.CreatedAt).To(BeTemporally("~", psModel.CreatedAt))
	Expect(*psOutput.Kind).To(Equal("ProjectSettings"))
	Expect(*psOutput.Href).To(Equal(fmt.Sprintf("/api/ambient-api-server/v1/project_settings/%s", *psOutput.Id)))

	jwtToken := ctx.Value(openapi.ContextAccessToken)
	restyResp, _ := resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		SetBody(`{ this is invalid }`).
		Patch(h.RestURL("/project_settings/foo"))

	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest))
}

func TestProjectSettingsPaging(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	_, err := newProjectSettingsList("Bronto", 20)
	Expect(err).NotTo(HaveOccurred())

	list, _, err := client.DefaultAPI.ApiAmbientApiServerV1ProjectSettingsGet(ctx).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting project settings list: %v", err)
	Expect(len(list.Items)).To(Equal(20))
	Expect(list.Size).To(Equal(int32(20)))
	Expect(list.Total).To(Equal(int32(20)))
	Expect(list.Page).To(Equal(int32(1)))

	list, _, err = client.DefaultAPI.ApiAmbientApiServerV1ProjectSettingsGet(ctx).Page(2).Size(5).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting project settings list: %v", err)
	Expect(len(list.Items)).To(Equal(5))
	Expect(list.Size).To(Equal(int32(5)))
	Expect(list.Total).To(Equal(int32(20)))
	Expect(list.Page).To(Equal(int32(2)))
}

func TestProjectSettingsListSearch(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	items, err := newProjectSettingsList("bronto", 20)
	Expect(err).NotTo(HaveOccurred())

	search := fmt.Sprintf("id in ('%s')", items[0].ID)
	list, _, err := client.DefaultAPI.ApiAmbientApiServerV1ProjectSettingsGet(ctx).Search(search).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting project settings list: %v", err)
	Expect(len(list.Items)).To(Equal(1))
	Expect(list.Total).To(Equal(int32(1)))
	Expect(*list.Items[0].Id).To(Equal(items[0].ID))
}
