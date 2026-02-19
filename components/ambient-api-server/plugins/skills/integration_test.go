package skills_test

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

func TestSkillGet(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	_, _, err := client.DefaultAPI.ApiAmbientApiServerV1SkillsIdGet(context.Background(), "foo").Execute()
	Expect(err).To(HaveOccurred(), "Expected 401 but got nil error")

	_, resp, err := client.DefaultAPI.ApiAmbientApiServerV1SkillsIdGet(ctx, "foo").Execute()
	Expect(err).To(HaveOccurred(), "Expected 404")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	skillModel, err := newSkill(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	skillOutput, resp, err := client.DefaultAPI.ApiAmbientApiServerV1SkillsIdGet(ctx, skillModel.ID).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	Expect(*skillOutput.Id).To(Equal(skillModel.ID), "found object does not match test object")
	Expect(*skillOutput.Kind).To(Equal("Skill"))
	Expect(*skillOutput.Href).To(Equal(fmt.Sprintf("/api/ambient-api-server/v1/skills/%s", skillModel.ID)))
	Expect(*skillOutput.CreatedAt).To(BeTemporally("~", skillModel.CreatedAt))
	Expect(*skillOutput.UpdatedAt).To(BeTemporally("~", skillModel.UpdatedAt))
}

func TestSkillPost(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	skillInput := openapi.Skill{
		Name:    "test-name",
		RepoUrl: openapi.PtrString("test-repo_url"),
		Prompt:  openapi.PtrString("test-prompt"),
	}

	skillOutput, resp, err := client.DefaultAPI.ApiAmbientApiServerV1SkillsPost(ctx).Skill(skillInput).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))
	Expect(*skillOutput.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(*skillOutput.Kind).To(Equal("Skill"))
	Expect(*skillOutput.Href).To(Equal(fmt.Sprintf("/api/ambient-api-server/v1/skills/%s", *skillOutput.Id)))

	jwtToken := ctx.Value(openapi.ContextAccessToken)
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		SetBody(`{ this is invalid }`).
		Post(h.RestURL("/skills"))

	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest))
}

func TestSkillPatch(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	skillModel, err := newSkill(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	skillOutput, resp, err := client.DefaultAPI.ApiAmbientApiServerV1SkillsIdPatch(ctx, skillModel.ID).SkillPatchRequest(openapi.SkillPatchRequest{}).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(*skillOutput.Id).To(Equal(skillModel.ID))
	Expect(*skillOutput.CreatedAt).To(BeTemporally("~", skillModel.CreatedAt))
	Expect(*skillOutput.Kind).To(Equal("Skill"))
	Expect(*skillOutput.Href).To(Equal(fmt.Sprintf("/api/ambient-api-server/v1/skills/%s", *skillOutput.Id)))

	jwtToken := ctx.Value(openapi.ContextAccessToken)
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		SetBody(`{ this is invalid }`).
		Patch(h.RestURL("/skills/foo"))

	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest))
}

func TestSkillPaging(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	_, err := newSkillList("Bronto", 20)
	Expect(err).NotTo(HaveOccurred())

	list, _, err := client.DefaultAPI.ApiAmbientApiServerV1SkillsGet(ctx).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting skill list: %v", err)
	Expect(len(list.Items)).To(Equal(20))
	Expect(list.Size).To(Equal(int32(20)))
	Expect(list.Total).To(Equal(int32(20)))
	Expect(list.Page).To(Equal(int32(1)))

	list, _, err = client.DefaultAPI.ApiAmbientApiServerV1SkillsGet(ctx).Page(2).Size(5).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting skill list: %v", err)
	Expect(len(list.Items)).To(Equal(5))
	Expect(list.Size).To(Equal(int32(5)))
	Expect(list.Total).To(Equal(int32(20)))
	Expect(list.Page).To(Equal(int32(2)))
}

func TestSkillListSearch(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	skills, err := newSkillList("bronto", 20)
	Expect(err).NotTo(HaveOccurred())

	search := fmt.Sprintf("id in ('%s')", skills[0].ID)
	list, _, err := client.DefaultAPI.ApiAmbientApiServerV1SkillsGet(ctx).Search(search).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting skill list: %v", err)
	Expect(len(list.Items)).To(Equal(1))
	Expect(list.Total).To(Equal(int32(1)))
	Expect(*list.Items[0].Id).To(Equal(skills[0].ID))
}
