package permissions_test

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

func TestPermissionGet(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	_, _, err := client.DefaultAPI.ApiAmbientApiServerV1PermissionsIdGet(context.Background(), "foo").Execute()
	Expect(err).To(HaveOccurred(), "Expected 401 but got nil error")

	_, resp, err := client.DefaultAPI.ApiAmbientApiServerV1PermissionsIdGet(ctx, "foo").Execute()
	Expect(err).To(HaveOccurred(), "Expected 404")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	permissionModel, err := newPermission(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	permissionOutput, resp, err := client.DefaultAPI.ApiAmbientApiServerV1PermissionsIdGet(ctx, permissionModel.ID).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	Expect(*permissionOutput.Id).To(Equal(permissionModel.ID), "found object does not match test object")
	Expect(*permissionOutput.Kind).To(Equal("Permission"))
	Expect(*permissionOutput.Href).To(Equal(fmt.Sprintf("/api/ambient-api-server/v1/permissions/%s", permissionModel.ID)))
	Expect(*permissionOutput.CreatedAt).To(BeTemporally("~", permissionModel.CreatedAt))
	Expect(*permissionOutput.UpdatedAt).To(BeTemporally("~", permissionModel.UpdatedAt))
}

func TestPermissionPost(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	permissionInput := openapi.Permission{
		SubjectType: "user",
		SubjectName: "test-user",
		Role:        "edit",
		ProjectId:   openapi.PtrString("test-project_id"),
	}

	permissionOutput, resp, err := client.DefaultAPI.ApiAmbientApiServerV1PermissionsPost(ctx).Permission(permissionInput).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))
	Expect(*permissionOutput.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(*permissionOutput.Kind).To(Equal("Permission"))
	Expect(*permissionOutput.Href).To(Equal(fmt.Sprintf("/api/ambient-api-server/v1/permissions/%s", *permissionOutput.Id)))

	jwtToken := ctx.Value(openapi.ContextAccessToken)
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		SetBody(`{ this is invalid }`).
		Post(h.RestURL("/permissions"))

	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest))
}

func TestPermissionPatch(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	permissionModel, err := newPermission(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	permissionOutput, resp, err := client.DefaultAPI.ApiAmbientApiServerV1PermissionsIdPatch(ctx, permissionModel.ID).PermissionPatchRequest(openapi.PermissionPatchRequest{}).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(*permissionOutput.Id).To(Equal(permissionModel.ID))
	Expect(*permissionOutput.CreatedAt).To(BeTemporally("~", permissionModel.CreatedAt))
	Expect(*permissionOutput.Kind).To(Equal("Permission"))
	Expect(*permissionOutput.Href).To(Equal(fmt.Sprintf("/api/ambient-api-server/v1/permissions/%s", *permissionOutput.Id)))

	jwtToken := ctx.Value(openapi.ContextAccessToken)
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		SetBody(`{ this is invalid }`).
		Patch(h.RestURL("/permissions/foo"))

	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest))
}

func TestPermissionPaging(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	_, err := newPermissionList("Bronto", 20)
	Expect(err).NotTo(HaveOccurred())

	list, _, err := client.DefaultAPI.ApiAmbientApiServerV1PermissionsGet(ctx).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting permission list: %v", err)
	Expect(len(list.Items)).To(Equal(20))
	Expect(list.Size).To(Equal(int32(20)))
	Expect(list.Total).To(Equal(int32(20)))
	Expect(list.Page).To(Equal(int32(1)))

	list, _, err = client.DefaultAPI.ApiAmbientApiServerV1PermissionsGet(ctx).Page(2).Size(5).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting permission list: %v", err)
	Expect(len(list.Items)).To(Equal(5))
	Expect(list.Size).To(Equal(int32(5)))
	Expect(list.Total).To(Equal(int32(20)))
	Expect(list.Page).To(Equal(int32(2)))
}

func TestPermissionListSearch(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	permissions, err := newPermissionList("bronto", 20)
	Expect(err).NotTo(HaveOccurred())

	search := fmt.Sprintf("id in ('%s')", permissions[0].ID)
	list, _, err := client.DefaultAPI.ApiAmbientApiServerV1PermissionsGet(ctx).Search(search).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting permission list: %v", err)
	Expect(len(list.Items)).To(Equal(1))
	Expect(list.Total).To(Equal(int32(1)))
	Expect(*list.Items[0].Id).To(Equal(permissions[0].ID))
}
