package users_test

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

func TestUserGet(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	_, _, err := client.DefaultAPI.ApiAmbientV1UsersIdGet(context.Background(), "foo").Execute()
	Expect(err).To(HaveOccurred(), "Expected 401 but got nil error")

	_, resp, err := client.DefaultAPI.ApiAmbientV1UsersIdGet(ctx, "foo").Execute()
	Expect(err).To(HaveOccurred(), "Expected 404")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	userModel, err := newUser(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	userOutput, resp, err := client.DefaultAPI.ApiAmbientV1UsersIdGet(ctx, userModel.ID).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	Expect(*userOutput.Id).To(Equal(userModel.ID), "found object does not match test object")
	Expect(*userOutput.Kind).To(Equal("User"))
	Expect(*userOutput.Href).To(Equal(fmt.Sprintf("/api/ambient/v1/users/%s", userModel.ID)))
	Expect(*userOutput.CreatedAt).To(BeTemporally("~", userModel.CreatedAt))
	Expect(*userOutput.UpdatedAt).To(BeTemporally("~", userModel.UpdatedAt))
}

func TestUserPost(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	userInput := openapi.User{
		Username: "test-username",
		Name:     "test-name",
		Email:    openapi.PtrString("test-email"),
	}

	userOutput, resp, err := client.DefaultAPI.ApiAmbientV1UsersPost(ctx).User(userInput).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))
	Expect(*userOutput.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(*userOutput.Kind).To(Equal("User"))
	Expect(*userOutput.Href).To(Equal(fmt.Sprintf("/api/ambient/v1/users/%s", *userOutput.Id)))

	jwtToken := ctx.Value(openapi.ContextAccessToken)
	restyResp, _ := resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		SetBody(`{ this is invalid }`).
		Post(h.RestURL("/users"))

	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest))
}

func TestUserPatch(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	userModel, err := newUser(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	userOutput, resp, err := client.DefaultAPI.ApiAmbientV1UsersIdPatch(ctx, userModel.ID).UserPatchRequest(openapi.UserPatchRequest{}).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(*userOutput.Id).To(Equal(userModel.ID))
	Expect(*userOutput.CreatedAt).To(BeTemporally("~", userModel.CreatedAt))
	Expect(*userOutput.Kind).To(Equal("User"))
	Expect(*userOutput.Href).To(Equal(fmt.Sprintf("/api/ambient/v1/users/%s", *userOutput.Id)))

	jwtToken := ctx.Value(openapi.ContextAccessToken)
	restyResp, _ := resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		SetBody(`{ this is invalid }`).
		Patch(h.RestURL("/users/foo"))

	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest))
}

func TestUserPaging(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	_, err := newUserList("Bronto", 20)
	Expect(err).NotTo(HaveOccurred())

	list, _, err := client.DefaultAPI.ApiAmbientV1UsersGet(ctx).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting user list: %v", err)
	Expect(len(list.Items)).To(Equal(20))
	Expect(list.Size).To(Equal(int32(20)))
	Expect(list.Total).To(Equal(int32(20)))
	Expect(list.Page).To(Equal(int32(1)))

	list, _, err = client.DefaultAPI.ApiAmbientV1UsersGet(ctx).Page(2).Size(5).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting user list: %v", err)
	Expect(len(list.Items)).To(Equal(5))
	Expect(list.Size).To(Equal(int32(5)))
	Expect(list.Total).To(Equal(int32(20)))
	Expect(list.Page).To(Equal(int32(2)))
}

func TestUserListSearch(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	users, err := newUserList("bronto", 20)
	Expect(err).NotTo(HaveOccurred())

	search := fmt.Sprintf("id in ('%s')", users[0].ID)
	list, _, err := client.DefaultAPI.ApiAmbientV1UsersGet(ctx).Search(search).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting user list: %v", err)
	Expect(len(list.Items)).To(Equal(1))
	Expect(list.Total).To(Equal(int32(1)))
	Expect(*list.Items[0].Id).To(Equal(users[0].ID))
}
