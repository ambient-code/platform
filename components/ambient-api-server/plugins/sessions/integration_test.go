package sessions_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"gopkg.in/resty.v1"

	"github.com/ambient/platform/components/ambient-api-server/plugins/sessions"
	"github.com/ambient/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/ambient/platform/components/ambient-api-server/test"
	"github.com/openshift-online/rh-trex-ai/pkg/environments"
)

func TestSessionGet(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	_, _, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsIdGet(context.Background(), "foo").Execute()
	Expect(err).To(HaveOccurred(), "Expected 401 but got nil error")

	_, resp, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsIdGet(ctx, "foo").Execute()
	Expect(err).To(HaveOccurred(), "Expected 404")
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))

	sessionModel, err := newSession(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	sessionOutput, resp, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsIdGet(ctx, sessionModel.ID).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	Expect(*sessionOutput.Id).To(Equal(sessionModel.ID), "found object does not match test object")
	Expect(*sessionOutput.Kind).To(Equal("Session"))
	Expect(*sessionOutput.Href).To(Equal(fmt.Sprintf("/api/ambient-api-server/v1/sessions/%s", sessionModel.ID)))
	Expect(*sessionOutput.CreatedAt).To(BeTemporally("~", sessionModel.CreatedAt))
	Expect(*sessionOutput.UpdatedAt).To(BeTemporally("~", sessionModel.UpdatedAt))
}

func TestSessionPost(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	creator, err := newUser("test-creator-post")
	Expect(err).NotTo(HaveOccurred(), "Error creating creator user")
	assignee, err := newUser("test-assignee-post")
	Expect(err).NotTo(HaveOccurred(), "Error creating assignee user")
	wf, err := newParentWorkflow()
	Expect(err).NotTo(HaveOccurred(), "Error creating parent workflow")

	sessionInput := openapi.Session{
		Name:            "test-name",
		RepoUrl:         openapi.PtrString("test-repo_url"),
		Prompt:          openapi.PtrString("test-prompt"),
		CreatedByUserId: openapi.PtrString(creator.ID),
		AssignedUserId:  openapi.PtrString(assignee.ID),
		WorkflowId:      openapi.PtrString(wf.ID),
	}

	sessionOutput, resp, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsPost(ctx).Session(sessionInput).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))
	Expect(*sessionOutput.Id).NotTo(BeEmpty(), "Expected ID assigned on creation")
	Expect(*sessionOutput.Kind).To(Equal("Session"))
	Expect(*sessionOutput.Href).To(Equal(fmt.Sprintf("/api/ambient-api-server/v1/sessions/%s", *sessionOutput.Id)))
	Expect(sessionOutput.CreatedByUserId).To(BeNil(), "created_by_user_id must be ignored on create (read-only field)")

	jwtToken := ctx.Value(openapi.ContextAccessToken)
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		SetBody(`{ this is invalid }`).
		Post(h.RestURL("/sessions"))

	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest))
}

func TestSessionPatch(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	sessionModel, err := newSession(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	sessionOutput, resp, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsIdPatch(ctx, sessionModel.ID).SessionPatchRequest(openapi.SessionPatchRequest{}).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error posting object:  %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(*sessionOutput.Id).To(Equal(sessionModel.ID))
	Expect(*sessionOutput.CreatedAt).To(BeTemporally("~", sessionModel.CreatedAt))
	Expect(*sessionOutput.Kind).To(Equal("Session"))
	Expect(*sessionOutput.Href).To(Equal(fmt.Sprintf("/api/ambient-api-server/v1/sessions/%s", *sessionOutput.Id)))

	jwtToken := ctx.Value(openapi.ContextAccessToken)
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		SetBody(`{ this is invalid }`).
		Patch(h.RestURL("/sessions/foo"))

	Expect(restyResp.StatusCode()).To(Equal(http.StatusBadRequest))
}

func TestSessionPaging(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	_, err := newSessionList("Bronto", 20)
	Expect(err).NotTo(HaveOccurred())

	list, _, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsGet(ctx).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting session list: %v", err)
	Expect(len(list.Items)).To(Equal(20))
	Expect(list.Size).To(Equal(int32(20)))
	Expect(list.Total).To(Equal(int32(20)))
	Expect(list.Page).To(Equal(int32(1)))

	list, _, err = client.DefaultAPI.ApiAmbientApiServerV1SessionsGet(ctx).Page(2).Size(5).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting session list: %v", err)
	Expect(len(list.Items)).To(Equal(5))
	Expect(list.Size).To(Equal(int32(5)))
	Expect(list.Total).To(Equal(int32(20)))
	Expect(list.Page).To(Equal(int32(2)))
}

func TestSessionExpandedFields(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	creator, err := newUser("test-creator-expanded")
	Expect(err).NotTo(HaveOccurred())
	wf, err := newParentWorkflow()
	Expect(err).NotTo(HaveOccurred())

	sessionInput := openapi.Session{
		Name:                 "expanded-session",
		Prompt:               openapi.PtrString("do something"),
		CreatedByUserId:      openapi.PtrString(creator.ID),
		WorkflowId:           openapi.PtrString(wf.ID),
		Repos:                openapi.PtrString(`[{"url":"https://github.com/test/repo","branch":"main"}]`),
		Interactive:          openapi.PtrBool(true),
		Timeout:              openapi.PtrInt32(3600),
		LlmModel:             openapi.PtrString("claude-sonnet-4-20250514"),
		LlmTemperature:       openapi.PtrFloat64(0.7),
		LlmMaxTokens:         openapi.PtrInt32(4096),
		BotAccountName:       openapi.PtrString("ambient-bot"),
		ResourceOverrides:    openapi.PtrString(`{"cpu":"2","memory":"4Gi"}`),
		EnvironmentVariables: openapi.PtrString(`{"FOO":"bar"}`),
		Labels:               openapi.PtrString(`{"env":"test"}`),
		Annotations:          openapi.PtrString(`{"owner":"ci"}`),
	}

	created, resp, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsPost(ctx).Session(sessionInput).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error creating expanded session: %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))

	Expect(*created.Repos).To(Equal(`[{"url":"https://github.com/test/repo","branch":"main"}]`))
	Expect(*created.Interactive).To(BeTrue())
	Expect(*created.Timeout).To(Equal(int32(3600)))
	Expect(*created.LlmModel).To(Equal("claude-sonnet-4-20250514"))
	Expect(*created.LlmTemperature).To(BeNumerically("~", 0.7, 0.001))
	Expect(*created.LlmMaxTokens).To(Equal(int32(4096)))
	Expect(*created.BotAccountName).To(Equal("ambient-bot"))
	Expect(*created.ResourceOverrides).To(Equal(`{"cpu":"2","memory":"4Gi"}`))
	Expect(*created.EnvironmentVariables).To(Equal(`{"FOO":"bar"}`))
	Expect(*created.Labels).To(Equal(`{"env":"test"}`))
	Expect(*created.Annotations).To(Equal(`{"owner":"ci"}`))

	Expect(created.KubeCrName).NotTo(BeNil(), "kube_cr_name should be auto-set")
	Expect(*created.KubeCrName).To(Equal(*created.Id), "kube_cr_name should equal session ID")

	Expect(created.CreatedByUserId).To(BeNil(), "created_by_user_id must be ignored on create (read-only)")
	Expect(created.Phase).To(BeNil(), "phase should be nil on creation")
	Expect(created.StartTime).To(BeNil(), "start_time should be nil on creation")

	fetched, resp, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsIdGet(ctx, *created.Id).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(*fetched.Interactive).To(BeTrue())
	Expect(*fetched.LlmModel).To(Equal("claude-sonnet-4-20250514"))
	Expect(*fetched.KubeCrName).To(Equal(*created.Id))

	patchReq := openapi.SessionPatchRequest{
		LlmModel: openapi.PtrString("claude-opus-4-20250514"),
		Timeout:  openapi.PtrInt32(7200),
	}
	patched, resp, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsIdPatch(ctx, *created.Id).SessionPatchRequest(patchReq).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(*patched.LlmModel).To(Equal("claude-opus-4-20250514"))
	Expect(*patched.Timeout).To(Equal(int32(7200)))
	Expect(*patched.Interactive).To(BeTrue(), "interactive should be unchanged after patch")
}

func TestSessionParentChild(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	parent, err := newSession(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	childInput := openapi.Session{
		Name:            "child-session",
		Prompt:          openapi.PtrString("child prompt"),
		ParentSessionId: openapi.PtrString(parent.ID),
	}

	child, resp, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsPost(ctx).Session(childInput).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))
	Expect(*child.ParentSessionId).To(Equal(parent.ID))

	fetched, resp, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsIdGet(ctx, *child.Id).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(*fetched.ParentSessionId).To(Equal(parent.ID))
}

func TestSessionStatusPatch(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	sessionModel, err := newSession(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	statusPatch := openapi.SessionStatusPatchRequest{
		Phase: openapi.PtrString("Running"),
	}
	patched, resp, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsIdStatusPatch(ctx, sessionModel.ID).SessionStatusPatchRequest(statusPatch).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error patching session status: %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(*patched.Phase).To(Equal("Running"))

	fetched, resp, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsIdGet(ctx, sessionModel.ID).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(*fetched.Phase).To(Equal("Running"))
}

func TestSessionStatusPatchMultipleFields(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	sessionModel, err := newSession(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	now := time.Now().UTC().Truncate(time.Millisecond)
	statusPatch := openapi.SessionStatusPatchRequest{
		Phase:        openapi.PtrString("Running"),
		StartTime:    &now,
		SdkSessionId: openapi.PtrString("sdk-abc-123"),
		Conditions:   openapi.PtrString(`[{"type":"Ready","status":"True"}]`),
		KubeNamespace: openapi.PtrString("ambient-code"),
		KubeCrUid:    openapi.PtrString("uid-xyz-456"),
	}
	patched, resp, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsIdStatusPatch(ctx, sessionModel.ID).SessionStatusPatchRequest(statusPatch).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error patching session status: %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(*patched.Phase).To(Equal("Running"))
	Expect(*patched.SdkSessionId).To(Equal("sdk-abc-123"))
	Expect(*patched.Conditions).To(Equal(`[{"type":"Ready","status":"True"}]`))
	Expect(*patched.KubeNamespace).To(Equal("ambient-code"))
	Expect(*patched.KubeCrUid).To(Equal("uid-xyz-456"))
	Expect(*patched.StartTime).To(BeTemporally("~", now, time.Second))
}

func TestSessionStatusPatchPreservesData(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	sessionInput := openapi.Session{
		Name:     "preserve-data-session",
		Prompt:   openapi.PtrString("important prompt"),
		LlmModel: openapi.PtrString("claude-sonnet-4-20250514"),
		Timeout:  openapi.PtrInt32(3600),
	}
	created, resp, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsPost(ctx).Session(sessionInput).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))

	statusPatch := openapi.SessionStatusPatchRequest{
		Phase: openapi.PtrString("Running"),
	}
	_, resp, err = client.DefaultAPI.ApiAmbientApiServerV1SessionsIdStatusPatch(ctx, *created.Id).SessionStatusPatchRequest(statusPatch).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	fetched, resp, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsIdGet(ctx, *created.Id).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(fetched.Name).To(Equal("preserve-data-session"))
	Expect(*fetched.Prompt).To(Equal("important prompt"))
	Expect(*fetched.LlmModel).To(Equal("claude-sonnet-4-20250514"))
	Expect(*fetched.Timeout).To(Equal(int32(3600)))
	Expect(*fetched.Phase).To(Equal("Running"))
}

func TestSessionStatusPatchNotFound(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	statusPatch := openapi.SessionStatusPatchRequest{
		Phase: openapi.PtrString("Running"),
	}
	_, resp, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsIdStatusPatch(ctx, "nonexistent-id").SessionStatusPatchRequest(statusPatch).Execute()
	Expect(err).To(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
}

func TestSessionRegularPatchIgnoresStatus(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	sessionModel, err := newSession(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	statusPatch := openapi.SessionStatusPatchRequest{
		Phase: openapi.PtrString("Running"),
	}
	_, resp, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsIdStatusPatch(ctx, sessionModel.ID).SessionStatusPatchRequest(statusPatch).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	jwtToken := ctx.Value(openapi.ContextAccessToken)
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		SetBody(`{"phase":"Completed","name":"updated-name"}`).
		Patch(h.RestURL(fmt.Sprintf("/sessions/%s", sessionModel.ID)))
	Expect(err).NotTo(HaveOccurred())
	Expect(restyResp.StatusCode()).To(Equal(http.StatusOK))

	fetched, resp, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsIdGet(ctx, sessionModel.ID).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(fetched.Name).To(Equal("updated-name"))
	Expect(*fetched.Phase).To(Equal("Running"), "regular PATCH should not update phase")
}

func TestSessionStatusPatchAuth(t *testing.T) {
	_, client := test.RegisterIntegration(t)

	statusPatch := openapi.SessionStatusPatchRequest{
		Phase: openapi.PtrString("Running"),
	}
	_, _, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsIdStatusPatch(context.Background(), "some-id").SessionStatusPatchRequest(statusPatch).Execute()
	Expect(err).To(HaveOccurred(), "Expected 401 but got nil error")
}

func TestSessionStart(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	sessionModel, err := newSession(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	started, resp, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsIdStartPost(ctx, sessionModel.ID).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error starting session: %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(*started.Phase).To(Equal("Pending"))
	Expect(started.Interactive).NotTo(BeNil(), "interactive should be set by start")
	Expect(*started.Interactive).To(BeTrue(), "start should force interactive=true")

	fetched, resp, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsIdGet(ctx, sessionModel.ID).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(*fetched.Phase).To(Equal("Pending"))
	Expect(*fetched.Interactive).To(BeTrue(), "interactive should persist after start")
}

func TestSessionStop(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	sessionModel, err := newSession(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	statusPatch := openapi.SessionStatusPatchRequest{
		Phase: openapi.PtrString("Running"),
	}
	_, resp, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsIdStatusPatch(ctx, sessionModel.ID).SessionStatusPatchRequest(statusPatch).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	stopped, resp, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsIdStopPost(ctx, sessionModel.ID).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error stopping session: %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(*stopped.Phase).To(Equal("Stopping"))

	fetched, resp, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsIdGet(ctx, sessionModel.ID).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(*fetched.Phase).To(Equal("Stopping"))
}

func TestSessionStartAlreadyRunning(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	sessionModel, err := newSession(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	statusPatch := openapi.SessionStatusPatchRequest{
		Phase: openapi.PtrString("Running"),
	}
	_, resp, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsIdStatusPatch(ctx, sessionModel.ID).SessionStatusPatchRequest(statusPatch).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	_, resp, err = client.DefaultAPI.ApiAmbientApiServerV1SessionsIdStartPost(ctx, sessionModel.ID).Execute()
	Expect(err).To(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusConflict))
}

func TestSessionStopAlreadyStopped(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	sessionModel, err := newSession(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	statusPatch := openapi.SessionStatusPatchRequest{
		Phase: openapi.PtrString("Stopped"),
	}
	_, resp, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsIdStatusPatch(ctx, sessionModel.ID).SessionStatusPatchRequest(statusPatch).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	_, resp, err = client.DefaultAPI.ApiAmbientApiServerV1SessionsIdStopPost(ctx, sessionModel.ID).Execute()
	Expect(err).To(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusConflict))
}

func TestSessionStartFromFailed(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	sessionModel, err := newSession(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	statusPatch := openapi.SessionStatusPatchRequest{
		Phase: openapi.PtrString("Failed"),
	}
	_, resp, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsIdStatusPatch(ctx, sessionModel.ID).SessionStatusPatchRequest(statusPatch).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	started, resp, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsIdStartPost(ctx, sessionModel.ID).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error starting session from Failed: %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(*started.Phase).To(Equal("Pending"))
}

func TestSessionStartFromCompleted(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	sessionModel, err := newSession(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	statusPatch := openapi.SessionStatusPatchRequest{
		Phase: openapi.PtrString("Completed"),
	}
	_, resp, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsIdStatusPatch(ctx, sessionModel.ID).SessionStatusPatchRequest(statusPatch).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	started, resp, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsIdStartPost(ctx, sessionModel.ID).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error starting session from Completed: %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(*started.Phase).To(Equal("Pending"))
}

func TestSessionStopFromPending(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	sessionModel, err := newSession(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	statusPatch := openapi.SessionStatusPatchRequest{
		Phase: openapi.PtrString("Pending"),
	}
	_, resp, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsIdStatusPatch(ctx, sessionModel.ID).SessionStatusPatchRequest(statusPatch).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	stopped, resp, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsIdStopPost(ctx, sessionModel.ID).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error stopping session from Pending: %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(*stopped.Phase).To(Equal("Stopping"))
}

func TestSessionStartNotFound(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	_, resp, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsIdStartPost(ctx, "nonexistent-id").Execute()
	Expect(err).To(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusNotFound))
}

func TestSessionStartAuth(t *testing.T) {
	_, client := test.RegisterIntegration(t)

	_, _, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsIdStartPost(context.Background(), "some-id").Execute()
	Expect(err).To(HaveOccurred(), "Expected 401 but got nil error")
}

func TestSessionLifecycle(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	sessionModel, err := newSession(h.NewID())
	Expect(err).NotTo(HaveOccurred())

	started, resp, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsIdStartPost(ctx, sessionModel.ID).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(*started.Phase).To(Equal("Pending"))

	statusPatch := openapi.SessionStatusPatchRequest{
		Phase: openapi.PtrString("Running"),
	}
	_, resp, err = client.DefaultAPI.ApiAmbientApiServerV1SessionsIdStatusPatch(ctx, sessionModel.ID).SessionStatusPatchRequest(statusPatch).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	stopped, resp, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsIdStopPost(ctx, sessionModel.ID).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(*stopped.Phase).To(Equal("Stopping"))

	statusPatch = openapi.SessionStatusPatchRequest{
		Phase: openapi.PtrString("Stopped"),
	}
	_, resp, err = client.DefaultAPI.ApiAmbientApiServerV1SessionsIdStatusPatch(ctx, sessionModel.ID).SessionStatusPatchRequest(statusPatch).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	restarted, resp, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsIdStartPost(ctx, sessionModel.ID).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(*restarted.Phase).To(Equal("Pending"))

	statusPatch = openapi.SessionStatusPatchRequest{
		Phase: openapi.PtrString("Running"),
	}
	_, resp, err = client.DefaultAPI.ApiAmbientApiServerV1SessionsIdStatusPatch(ctx, sessionModel.ID).SessionStatusPatchRequest(statusPatch).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	statusPatch = openapi.SessionStatusPatchRequest{
		Phase: openapi.PtrString("Failed"),
	}
	_, resp, err = client.DefaultAPI.ApiAmbientApiServerV1SessionsIdStatusPatch(ctx, sessionModel.ID).SessionStatusPatchRequest(statusPatch).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))

	restartedFromFailed, resp, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsIdStartPost(ctx, sessionModel.ID).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(*restartedFromFailed.Phase).To(Equal("Pending"))
}

func TestSessionLlmDefaults(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	sessionInput := openapi.Session{
		Name:   "no-llm-session",
		Prompt: openapi.PtrString("test prompt"),
	}

	created, resp, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsPost(ctx).Session(sessionInput).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error creating session without LLM fields: %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))

	Expect(created.LlmModel).NotTo(BeNil(), "llm_model should be defaulted")
	Expect(*created.LlmModel).To(Equal("sonnet"))
	Expect(created.LlmTemperature).NotTo(BeNil(), "llm_temperature should be defaulted")
	Expect(*created.LlmTemperature).To(BeNumerically("~", 0.7, 0.001))
	Expect(created.LlmMaxTokens).NotTo(BeNil(), "llm_max_tokens should be defaulted")
	Expect(*created.LlmMaxTokens).To(Equal(int32(4000)))

	fetched, resp, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsIdGet(ctx, *created.Id).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(*fetched.LlmModel).To(Equal("sonnet"))
	Expect(*fetched.LlmTemperature).To(BeNumerically("~", 0.7, 0.001))
	Expect(*fetched.LlmMaxTokens).To(Equal(int32(4000)))
}

func TestSessionLlmDefaultsPreservedWhenProvided(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	sessionInput := openapi.Session{
		Name:           "custom-llm-session",
		Prompt:         openapi.PtrString("test prompt"),
		LlmModel:       openapi.PtrString("claude-opus-4-20250514"),
		LlmTemperature: openapi.PtrFloat64(0.3),
		LlmMaxTokens:   openapi.PtrInt32(8000),
	}

	created, resp, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsPost(ctx).Session(sessionInput).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error creating session with custom LLM fields: %v", err)
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))

	Expect(*created.LlmModel).To(Equal("claude-opus-4-20250514"))
	Expect(*created.LlmTemperature).To(BeNumerically("~", 0.3, 0.001))
	Expect(*created.LlmMaxTokens).To(Equal(int32(8000)))
}

func TestSessionCreatedByUserIdReadOnly(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	sessionInput := openapi.Session{
		Name:            "readonly-test",
		CreatedByUserId: openapi.PtrString("attacker-injected-user-id"),
	}
	created, resp, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsPost(ctx).Session(sessionInput).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusCreated))
	Expect(created.CreatedByUserId).To(BeNil(), "created_by_user_id must not be settable via POST")

	jwtToken := ctx.Value(openapi.ContextAccessToken)
	restyResp, err := resty.R().
		SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", jwtToken)).
		SetBody(`{"created_by_user_id":"attacker-injected-via-patch","name":"patched-name"}`).
		Patch(h.RestURL(fmt.Sprintf("/sessions/%s", *created.Id)))
	Expect(err).NotTo(HaveOccurred())
	Expect(restyResp.StatusCode()).To(Equal(http.StatusOK))

	fetched, resp, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsIdGet(ctx, *created.Id).Execute()
	Expect(err).NotTo(HaveOccurred())
	Expect(resp.StatusCode).To(Equal(http.StatusOK))
	Expect(fetched.Name).To(Equal("patched-name"), "name should be updated by PATCH")
	Expect(fetched.CreatedByUserId).To(BeNil(), "created_by_user_id must not be settable via PATCH")
}

func TestSessionAll(t *testing.T) {
	h, _ := test.RegisterIntegration(t)
	_ = h

	created, err := newSessionList("all", 5)
	Expect(err).NotTo(HaveOccurred())

	sessionService := sessions.Service(&environments.Environment().Services)
	all, svcErr := sessionService.All(context.Background())
	Expect(svcErr).NotTo(HaveOccurred(), "Error calling All(): %v", svcErr)
	Expect(len(all)).To(Equal(5))

	returnedIDs := map[string]bool{}
	for _, s := range all {
		returnedIDs[s.ID] = true
	}
	for _, s := range created {
		Expect(returnedIDs).To(HaveKey(s.ID), "All() should return session %s", s.ID)
	}
}

func TestSessionAllEmpty(t *testing.T) {
	h, _ := test.RegisterIntegration(t)
	_ = h

	sessionService := sessions.Service(&environments.Environment().Services)
	all, svcErr := sessionService.All(context.Background())
	Expect(svcErr).NotTo(HaveOccurred(), "Error calling All() on empty table: %v", svcErr)
	Expect(len(all)).To(Equal(0))
}

func TestSessionListSearch(t *testing.T) {
	h, client := test.RegisterIntegration(t)

	account := h.NewRandAccount()
	ctx := h.NewAuthenticatedContext(account)

	sessions, err := newSessionList("bronto", 20)
	Expect(err).NotTo(HaveOccurred())

	search := fmt.Sprintf("id in ('%s')", sessions[0].ID)
	list, _, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsGet(ctx).Search(search).Execute()
	Expect(err).NotTo(HaveOccurred(), "Error getting session list: %v", err)
	Expect(len(list.Items)).To(Equal(1))
	Expect(list.Total).To(Equal(int32(1)))
	Expect(*list.Items[0].Id).To(Equal(sessions[0].ID))
}
