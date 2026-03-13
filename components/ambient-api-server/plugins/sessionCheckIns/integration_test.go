package sessionCheckIns_test

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

func TestSessionCheckInGet(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	_, _, err := client.DefaultAPI.ApiAmbientV1SessionCheckInsIdGet(context.Background(), "foo").Execute()
	Expect(err).To(HaveOccurred(), "Expected 401 but got nil error")

	_, resp, err := client.DefaultAPI.ApiAmbientV1SessionCheckInsIdGet(ctx, "foo").Execute()
	Expect(err).To(HaveOccurred(), "Expected 404")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	sessionCheckInModel, err := newSessionCheckIn(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	sessionCheckInOutput, resp, err := client.DefaultAPI.ApiAmbientV1SessionCheckInsIdGet(ctx, sessionCheckInModel.ID).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	Expect(*sessionCheckInOutput.Id).To(Equal(sessionCheckInModel.ID), "found object does not match test object")
	Expect(*sessionCheckInOutput.Kind).To(Equal("SessionCheckIn"))
	Expect(*sessionCheckInOutput.Href).To(Equal(fmt.Sprintf("/api/ambient/v1/session_check_ins/%s", sessionCheckInModel.ID)))
	Expect(*sessionCheckInOutput.CreatedAt).To(BeTemporally("~", sessionCheckInModel.CreatedAt))
	Expect(*sessionCheckInOutput.UpdatedAt).To(BeTemporally("~", sessionCheckInModel.UpdatedAt))
}

func TestSessionCheckInPost(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	sessionCheckInInput := openapi.SessionCheckIn{
		SessionId: "test-session_id",
		AgentId:   "test-agent_id",
		Summary:   openapi.PtrString("test-summary"),
		Branch:    openapi.PtrString("test-branch"),
		Worktree:  openapi.PtrString("test-worktree"),
		Pr:        openapi.PtrString("test-pr"),
		Phase:     openapi.PtrString("test-phase"),
		TestCount: openapi.PtrInt32(42),
		NextSteps: openapi.PtrString("test-next_steps"),
	}

	sessionCheckInOutput, resp, err := client.DefaultAPI.ApiAmbientV1SessionCheckInsPost(ctx).SessionCheckIn(sessionCheckInInput).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))
	Expect(*sessionCheckInOutput.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(*sessionCheckInOutput.Kind).To(Equal("SessionCheckIn"))
	Expect(*sessionCheckInOutput.Href).To(Equal(fmt.Sprintf("/api/ambient/v1/session_check_ins/%s", *sessionCheckInOutput.Id)))

	jwtToken := ctx.Value(openapi.ContextAccessToken)
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		SetBody(`{ this is invalid }`).
		Post(h.RestURL("/session_check_ins"))

	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest))
}

func TestSessionCheckInPatch(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	sessionCheckInModel, err := newSessionCheckIn(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	sessionCheckInOutput, resp, err := client.DefaultAPI.ApiAmbientV1SessionCheckInsIdPatch(ctx, sessionCheckInModel.ID).SessionCheckInPatchRequest(openapi.SessionCheckInPatchRequest{}).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(*sessionCheckInOutput.Id).To(Equal(sessionCheckInModel.ID))
	Expect(*sessionCheckInOutput.CreatedAt).To(BeTemporally("~", sessionCheckInModel.CreatedAt))
	Expect(*sessionCheckInOutput.Kind).To(Equal("SessionCheckIn"))
	Expect(*sessionCheckInOutput.Href).To(Equal(fmt.Sprintf("/api/ambient/v1/session_check_ins/%s", *sessionCheckInOutput.Id)))

	jwtToken := ctx.Value(openapi.ContextAccessToken)
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		SetBody(`{ this is invalid }`).
		Patch(h.RestURL("/session_check_ins/foo"))

	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest))
}

func TestSessionCheckInPaging(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	_, err := newSessionCheckInList("Bronto", 20)
	Expect(err).NotTo(HaveOccurred())

	list, _, err := client.DefaultAPI.ApiAmbientV1SessionCheckInsGet(ctx).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting sessionCheckIn list: %v", err)
	Expect(len(list.Items)).To(Equal(20))
	Expect(list.Size).To(Equal(int32(20)))
	Expect(list.Total).To(Equal(int32(20)))
	Expect(list.Page).To(Equal(int32(1)))

	list, _, err = client.DefaultAPI.ApiAmbientV1SessionCheckInsGet(ctx).Page(2).Size(5).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting sessionCheckIn list: %v", err)
	Expect(len(list.Items)).To(Equal(5))
	Expect(list.Size).To(Equal(int32(5)))
	Expect(list.Total).To(Equal(int32(20)))
	Expect(list.Page).To(Equal(int32(2)))
}

func TestSessionCheckInListSearch(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	sessionCheckIns, err := newSessionCheckInList("bronto", 20)
	Expect(err).NotTo(HaveOccurred())

	search := fmt.Sprintf("id in ('%s')", sessionCheckIns[0].ID)
	list, _, err := client.DefaultAPI.ApiAmbientV1SessionCheckInsGet(ctx).Search(search).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting sessionCheckIn list: %v", err)
	Expect(len(list.Items)).To(Equal(1))
	Expect(list.Total).To(Equal(int32(1)))
	Expect(*list.Items[0].Id).To(Equal(sessionCheckIns[0].ID))
}
