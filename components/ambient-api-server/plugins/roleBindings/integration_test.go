package roleBindings_test

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

func TestRoleBindingGet(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	_, _, err := client.DefaultAPI.ApiAmbientV1RoleBindingsIdGet(context.Background(), "foo").Execute()
	Expect(err).To(HaveOccurred(), "Expected 401 but got nil error")

	_, resp, err := client.DefaultAPI.ApiAmbientV1RoleBindingsIdGet(ctx, "foo").Execute()
	Expect(err).To(HaveOccurred(), "Expected 404")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	roleBindingModel, err := newRoleBinding(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	roleBindingOutput, resp, err := client.DefaultAPI.ApiAmbientV1RoleBindingsIdGet(ctx, roleBindingModel.ID).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	Expect(*roleBindingOutput.Id).To(Equal(roleBindingModel.ID), "found object does not match test object")
	Expect(*roleBindingOutput.Kind).To(Equal("RoleBinding"))
	Expect(*roleBindingOutput.Href).To(Equal(fmt.Sprintf("/api/ambient/v1/role_bindings/%s", roleBindingModel.ID)))
	Expect(*roleBindingOutput.CreatedAt).To(BeTemporally("~", roleBindingModel.CreatedAt))
	Expect(*roleBindingOutput.UpdatedAt).To(BeTemporally("~", roleBindingModel.UpdatedAt))
}

func TestRoleBindingPost(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	roleBindingInput := openapi.RoleBinding{
		UserId:  "test-user_id",
		RoleId:  "test-role_id",
		Scope:   "test-scope",
		ScopeId: openapi.PtrString("test-scope_id"),
	}

	roleBindingOutput, resp, err := client.DefaultAPI.ApiAmbientV1RoleBindingsPost(ctx).RoleBinding(roleBindingInput).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))
	Expect(*roleBindingOutput.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(*roleBindingOutput.Kind).To(Equal("RoleBinding"))
	Expect(*roleBindingOutput.Href).To(Equal(fmt.Sprintf("/api/ambient/v1/role_bindings/%s", *roleBindingOutput.Id)))

	jwtToken := ctx.Value(openapi.ContextAccessToken)
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		SetBody(`{ this is invalid }`).
		Post(h.RestURL("/role_bindings"))

	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest))
}

func TestRoleBindingPatch(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	roleBindingModel, err := newRoleBinding(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	roleBindingOutput, resp, err := client.DefaultAPI.ApiAmbientV1RoleBindingsIdPatch(ctx, roleBindingModel.ID).RoleBindingPatchRequest(openapi.RoleBindingPatchRequest{}).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(*roleBindingOutput.Id).To(Equal(roleBindingModel.ID))
	Expect(*roleBindingOutput.CreatedAt).To(BeTemporally("~", roleBindingModel.CreatedAt))
	Expect(*roleBindingOutput.Kind).To(Equal("RoleBinding"))
	Expect(*roleBindingOutput.Href).To(Equal(fmt.Sprintf("/api/ambient/v1/role_bindings/%s", *roleBindingOutput.Id)))

	jwtToken := ctx.Value(openapi.ContextAccessToken)
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		SetBody(`{ this is invalid }`).
		Patch(h.RestURL("/role_bindings/foo"))

	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest))
}

func TestRoleBindingPaging(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	_, err := newRoleBindingList("Bronto", 20)
	Expect(err).NotTo(HaveOccurred())

	list, _, err := client.DefaultAPI.ApiAmbientV1RoleBindingsGet(ctx).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting roleBinding list: %v", err)
	Expect(len(list.Items)).To(Equal(20))
	Expect(list.Size).To(Equal(int32(20)))
	Expect(list.Total).To(Equal(int32(20)))
	Expect(list.Page).To(Equal(int32(1)))

	list, _, err = client.DefaultAPI.ApiAmbientV1RoleBindingsGet(ctx).Page(2).Size(5).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting roleBinding list: %v", err)
	Expect(len(list.Items)).To(Equal(5))
	Expect(list.Size).To(Equal(int32(5)))
	Expect(list.Total).To(Equal(int32(20)))
	Expect(list.Page).To(Equal(int32(2)))
}

func TestRoleBindingListSearch(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	roleBindings, err := newRoleBindingList("bronto", 20)
	Expect(err).NotTo(HaveOccurred())

	search := fmt.Sprintf("id in ('%s')", roleBindings[0].ID)
	list, _, err := client.DefaultAPI.ApiAmbientV1RoleBindingsGet(ctx).Search(search).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting roleBinding list: %v", err)
	Expect(len(list.Items)).To(Equal(1))
	Expect(list.Total).To(Equal(int32(1)))
	Expect(*list.Items[0].Id).To(Equal(roleBindings[0].ID))
}
