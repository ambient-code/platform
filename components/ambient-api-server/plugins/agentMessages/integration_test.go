package agentMessages_test

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

func TestAgentMessageGet(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	_, _, err := client.DefaultAPI.ApiAmbientV1AgentMessagesIdGet(context.Background(), "foo").Execute()
	Expect(err).To(HaveOccurred(), "Expected 401 but got nil error")

	_, resp, err := client.DefaultAPI.ApiAmbientV1AgentMessagesIdGet(ctx, "foo").Execute()
	Expect(err).To(HaveOccurred(), "Expected 404")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	agentMessageModel, err := newAgentMessage(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	agentMessageOutput, resp, err := client.DefaultAPI.ApiAmbientV1AgentMessagesIdGet(ctx, agentMessageModel.ID).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	Expect(*agentMessageOutput.Id).To(Equal(agentMessageModel.ID), "found object does not match test object")
	Expect(*agentMessageOutput.Kind).To(Equal("AgentMessage"))
	Expect(*agentMessageOutput.Href).To(Equal(fmt.Sprintf("/api/ambient/v1/agent_messages/%s", agentMessageModel.ID)))
	Expect(*agentMessageOutput.CreatedAt).To(BeTemporally("~", agentMessageModel.CreatedAt))
	Expect(*agentMessageOutput.UpdatedAt).To(BeTemporally("~", agentMessageModel.UpdatedAt))
}

func TestAgentMessagePost(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	agentMessageInput := openapi.AgentMessage{
		RecipientAgentId: "test-recipient_agent_id",
		SenderAgentId:    openapi.PtrString("test-sender_agent_id"),
		SenderUserId:     openapi.PtrString("test-sender_user_id"),
		SenderName:       openapi.PtrString("test-sender_name"),
		Body:             openapi.PtrString("test-body"),
		Read:             openapi.PtrBool(true),
	}

	agentMessageOutput, resp, err := client.DefaultAPI.ApiAmbientV1AgentMessagesPost(ctx).AgentMessage(agentMessageInput).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))
	Expect(*agentMessageOutput.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(*agentMessageOutput.Kind).To(Equal("AgentMessage"))
	Expect(*agentMessageOutput.Href).To(Equal(fmt.Sprintf("/api/ambient/v1/agent_messages/%s", *agentMessageOutput.Id)))

	jwtToken := ctx.Value(openapi.ContextAccessToken)
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		SetBody(`{ this is invalid }`).
		Post(h.RestURL("/agent_messages"))

	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest))
}

func TestAgentMessagePatch(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	agentMessageModel, err := newAgentMessage(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	agentMessageOutput, resp, err := client.DefaultAPI.ApiAmbientV1AgentMessagesIdPatch(ctx, agentMessageModel.ID).AgentMessagePatchRequest(openapi.AgentMessagePatchRequest{}).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(*agentMessageOutput.Id).To(Equal(agentMessageModel.ID))
	Expect(*agentMessageOutput.CreatedAt).To(BeTemporally("~", agentMessageModel.CreatedAt))
	Expect(*agentMessageOutput.Kind).To(Equal("AgentMessage"))
	Expect(*agentMessageOutput.Href).To(Equal(fmt.Sprintf("/api/ambient/v1/agent_messages/%s", *agentMessageOutput.Id)))

	jwtToken := ctx.Value(openapi.ContextAccessToken)
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		SetBody(`{ this is invalid }`).
		Patch(h.RestURL("/agent_messages/foo"))

	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest))
}

func TestAgentMessagePaging(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	_, err := newAgentMessageList("Bronto", 20)
	Expect(err).NotTo(HaveOccurred())

	list, _, err := client.DefaultAPI.ApiAmbientV1AgentMessagesGet(ctx).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting agentMessage list: %v", err)
	Expect(len(list.Items)).To(Equal(20))
	Expect(list.Size).To(Equal(int32(20)))
	Expect(list.Total).To(Equal(int32(20)))
	Expect(list.Page).To(Equal(int32(1)))

	list, _, err = client.DefaultAPI.ApiAmbientV1AgentMessagesGet(ctx).Page(2).Size(5).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting agentMessage list: %v", err)
	Expect(len(list.Items)).To(Equal(5))
	Expect(list.Size).To(Equal(int32(5)))
	Expect(list.Total).To(Equal(int32(20)))
	Expect(list.Page).To(Equal(int32(2)))
}

func TestAgentMessageListSearch(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	agentMessages, err := newAgentMessageList("bronto", 20)
	Expect(err).NotTo(HaveOccurred())

	search := fmt.Sprintf("id in ('%s')", agentMessages[0].ID)
	list, _, err := client.DefaultAPI.ApiAmbientV1AgentMessagesGet(ctx).Search(search).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting agentMessage list: %v", err)
	Expect(len(list.Items)).To(Equal(1))
	Expect(list.Total).To(Equal(int32(1)))
	Expect(*list.Items[0].Id).To(Equal(agentMessages[0].ID))
}
