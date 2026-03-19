package roles_test

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

func TestRoleGet(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	_, _, err := client.DefaultAPI.ApiAmbientV1RolesIdGet(context.Background(), "foo").Execute()
	Expect(err).To(HaveOccurred(), "Expected 401 but got nil error")

	_, resp, err := client.DefaultAPI.ApiAmbientV1RolesIdGet(ctx, "foo").Execute()
	Expect(err).To(HaveOccurred(), "Expected 404")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	roleModel, err := newRole(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	roleOutput, resp, err := client.DefaultAPI.ApiAmbientV1RolesIdGet(ctx, roleModel.ID).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	Expect(*roleOutput.Id).To(Equal(roleModel.ID), "found object does not match test object")
	Expect(*roleOutput.Kind).To(Equal("Role"))
	Expect(*roleOutput.Href).To(Equal(fmt.Sprintf("/api/ambient/v1/roles/%s", roleModel.ID)))
	Expect(*roleOutput.CreatedAt).To(BeTemporally("~", roleModel.CreatedAt))
	Expect(*roleOutput.UpdatedAt).To(BeTemporally("~", roleModel.UpdatedAt))
}

func TestRolePost(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	roleInput := openapi.Role{
		Name:        "test-name",
		DisplayName: openapi.PtrString("test-display_name"),
		Description: openapi.PtrString("test-description"),
		Permissions: openapi.PtrString("test-permissions"),
		BuiltIn:     openapi.PtrBool(true),
	}

	roleOutput, resp, err := client.DefaultAPI.ApiAmbientV1RolesPost(ctx).Role(roleInput).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))
	Expect(*roleOutput.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(*roleOutput.Kind).To(Equal("Role"))
	Expect(*roleOutput.Href).To(Equal(fmt.Sprintf("/api/ambient/v1/roles/%s", *roleOutput.Id)))

	jwtToken := ctx.Value(openapi.ContextAccessToken)
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		SetBody(`{ this is invalid }`).
		Post(h.RestURL("/roles"))

	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest))
}

func TestRolePatch(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	roleModel, err := newRole(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	roleOutput, resp, err := client.DefaultAPI.ApiAmbientV1RolesIdPatch(ctx, roleModel.ID).RolePatchRequest(openapi.RolePatchRequest{}).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(*roleOutput.Id).To(Equal(roleModel.ID))
	Expect(*roleOutput.CreatedAt).To(BeTemporally("~", roleModel.CreatedAt))
	Expect(*roleOutput.Kind).To(Equal("Role"))
	Expect(*roleOutput.Href).To(Equal(fmt.Sprintf("/api/ambient/v1/roles/%s", *roleOutput.Id)))

	jwtToken := ctx.Value(openapi.ContextAccessToken)
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		SetBody(`{ this is invalid }`).
		Patch(h.RestURL("/roles/foo"))

	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest))
}

func TestRolePaging(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	_, err := newRoleList("Bronto", 20)
	Expect(err).NotTo(HaveOccurred())

	list, _, err := client.DefaultAPI.ApiAmbientV1RolesGet(ctx).Size(20).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting role list: %v", err)
	Expect(len(list.Items)).To(Equal(20))
	Expect(list.Size).To(Equal(int32(20)))
	Expect(list.Total).To(BeNumerically(">=", int32(20)))
	Expect(list.Page).To(Equal(int32(1)))

	list, _, err = client.DefaultAPI.ApiAmbientV1RolesGet(ctx).Page(2).Size(5).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting role list: %v", err)
	Expect(len(list.Items)).To(Equal(5))
	Expect(list.Size).To(Equal(int32(5)))
	Expect(list.Total).To(BeNumerically(">=", int32(20)))
	Expect(list.Page).To(Equal(int32(2)))
}

func TestRoleListSearch(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	roles, err := newRoleList("bronto", 20)
	Expect(err).NotTo(HaveOccurred())

	search := fmt.Sprintf("id in ('%s')", roles[0].ID)
	list, _, err := client.DefaultAPI.ApiAmbientV1RolesGet(ctx).Search(search).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting role list: %v", err)
	Expect(len(list.Items)).To(Equal(1))
	Expect(list.Total).To(Equal(int32(1)))
	Expect(*list.Items[0].Id).To(Equal(roles[0].ID))
}
