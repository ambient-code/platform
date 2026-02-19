package workflowSkills_test

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

func TestWorkflowSkillGet(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	_, _, err := client.DefaultAPI.ApiAmbientApiServerV1WorkflowSkillsIdGet(context.Background(), "foo").Execute()
	Expect(err).To(HaveOccurred(), "Expected 401 but got nil error")

	_, resp, err := client.DefaultAPI.ApiAmbientApiServerV1WorkflowSkillsIdGet(ctx, "foo").Execute()
	Expect(err).To(HaveOccurred(), "Expected 404")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	workflowSkillModel, err := newWorkflowSkill(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	workflowSkillOutput, resp, err := client.DefaultAPI.ApiAmbientApiServerV1WorkflowSkillsIdGet(ctx, workflowSkillModel.ID).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	Expect(*workflowSkillOutput.Id).To(Equal(workflowSkillModel.ID), "found object does not match test object")
	Expect(*workflowSkillOutput.Kind).To(Equal("WorkflowSkill"))
	Expect(*workflowSkillOutput.Href).To(Equal(fmt.Sprintf("/api/ambient-api-server/v1/workflow_skills/%s", workflowSkillModel.ID)))
	Expect(*workflowSkillOutput.CreatedAt).To(BeTemporally("~", workflowSkillModel.CreatedAt))
	Expect(*workflowSkillOutput.UpdatedAt).To(BeTemporally("~", workflowSkillModel.UpdatedAt))
}

func TestWorkflowSkillPost(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	wf, err := newParentWorkflow()
	Expect(err).NotTo(HaveOccurred(), "Error creating parent workflow")
	sk, err := newParentSkill()
	Expect(err).NotTo(HaveOccurred(), "Error creating parent skill")

	workflowSkillInput := openapi.WorkflowSkill{
		WorkflowId: wf.ID,
		SkillId:    sk.ID,
		Position:   42,
	}

	workflowSkillOutput, resp, err := client.DefaultAPI.ApiAmbientApiServerV1WorkflowSkillsPost(ctx).WorkflowSkill(workflowSkillInput).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))
	Expect(*workflowSkillOutput.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(*workflowSkillOutput.Kind).To(Equal("WorkflowSkill"))
	Expect(*workflowSkillOutput.Href).To(Equal(fmt.Sprintf("/api/ambient-api-server/v1/workflow_skills/%s", *workflowSkillOutput.Id)))

	jwtToken := ctx.Value(openapi.ContextAccessToken)
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		SetBody(`{ this is invalid }`).
		Post(h.RestURL("/workflow_skills"))

	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest))
}

func TestWorkflowSkillPatch(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	workflowSkillModel, err := newWorkflowSkill(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	workflowSkillOutput, resp, err := client.DefaultAPI.ApiAmbientApiServerV1WorkflowSkillsIdPatch(ctx, workflowSkillModel.ID).WorkflowSkillPatchRequest(openapi.WorkflowSkillPatchRequest{}).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(*workflowSkillOutput.Id).To(Equal(workflowSkillModel.ID))
	Expect(*workflowSkillOutput.CreatedAt).To(BeTemporally("~", workflowSkillModel.CreatedAt))
	Expect(*workflowSkillOutput.Kind).To(Equal("WorkflowSkill"))
	Expect(*workflowSkillOutput.Href).To(Equal(fmt.Sprintf("/api/ambient-api-server/v1/workflow_skills/%s", *workflowSkillOutput.Id)))

	jwtToken := ctx.Value(openapi.ContextAccessToken)
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		SetBody(`{ this is invalid }`).
		Patch(h.RestURL("/workflow_skills/foo"))

	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest))
}

func TestWorkflowSkillPaging(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	_, err := newWorkflowSkillList("Bronto", 20)
	Expect(err).NotTo(HaveOccurred())

	list, _, err := client.DefaultAPI.ApiAmbientApiServerV1WorkflowSkillsGet(ctx).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting workflowSkill list: %v", err)
	Expect(len(list.Items)).To(Equal(20))
	Expect(list.Size).To(Equal(int32(20)))
	Expect(list.Total).To(Equal(int32(20)))
	Expect(list.Page).To(Equal(int32(1)))

	list, _, err = client.DefaultAPI.ApiAmbientApiServerV1WorkflowSkillsGet(ctx).Page(2).Size(5).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting workflowSkill list: %v", err)
	Expect(len(list.Items)).To(Equal(5))
	Expect(list.Size).To(Equal(int32(5)))
	Expect(list.Total).To(Equal(int32(20)))
	Expect(list.Page).To(Equal(int32(2)))
}

func TestWorkflowSkillListSearch(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	workflowSkills, err := newWorkflowSkillList("bronto", 20)
	Expect(err).NotTo(HaveOccurred())

	search := fmt.Sprintf("id in ('%s')", workflowSkills[0].ID)
	list, _, err := client.DefaultAPI.ApiAmbientApiServerV1WorkflowSkillsGet(ctx).Search(search).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting workflowSkill list: %v", err)
	Expect(len(list.Items)).To(Equal(1))
	Expect(list.Total).To(Equal(int32(1)))
	Expect(*list.Items[0].Id).To(Equal(workflowSkills[0].ID))
}
