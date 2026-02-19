package integration

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ambient/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/ambient/platform/components/ambient-api-server/test"
)

func TestSearchSessionsByProjectId(t *testing.T) {
	helper := test.NewHelper(t)
	client := helper.NewApiClient()
	account := helper.NewRandAccount()
	ctx := helper.NewAuthenticatedContext(account)

	projA, _, err := client.DefaultAPI.ApiAmbientApiServerV1ProjectsPost(ctx).Project(*openapi.NewProject("search-proj-a")).Execute()
	require.NoError(t, err)
	projB, _, err := client.DefaultAPI.ApiAmbientApiServerV1ProjectsPost(ctx).Project(*openapi.NewProject("search-proj-b")).Execute()
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		s := openapi.NewSession(fmt.Sprintf("proj-a-session-%d", i))
		s.SetProjectId(projA.GetId())
		_, _, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsPost(ctx).Session(*s).Execute()
		require.NoError(t, err)
	}
	for i := 0; i < 2; i++ {
		s := openapi.NewSession(fmt.Sprintf("proj-b-session-%d", i))
		s.SetProjectId(projB.GetId())
		_, _, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsPost(ctx).Session(*s).Execute()
		require.NoError(t, err)
	}

	search := fmt.Sprintf("project_id = '%s'", projA.GetId())
	list, _, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsGet(ctx).Search(search).Execute()
	require.NoError(t, err)
	assert.Equal(t, int32(3), list.Total)
	for _, item := range list.Items {
		assert.Equal(t, projA.GetId(), item.GetProjectId())
	}

	search = fmt.Sprintf("project_id = '%s'", projB.GetId())
	list, _, err = client.DefaultAPI.ApiAmbientApiServerV1SessionsGet(ctx).Search(search).Execute()
	require.NoError(t, err)
	assert.Equal(t, int32(2), list.Total)
	for _, item := range list.Items {
		assert.Equal(t, projB.GetId(), item.GetProjectId())
	}
}

func TestSearchSessionsByWorkflowId(t *testing.T) {
	helper := test.NewHelper(t)
	client := helper.NewApiClient()
	account := helper.NewRandAccount()
	ctx := helper.NewAuthenticatedContext(account)

	agent, _, err := client.DefaultAPI.ApiAmbientApiServerV1AgentsPost(ctx).Agent(*openapi.NewAgent("search-wf-agent")).Execute()
	require.NoError(t, err)

	wfA := openapi.NewWorkflow("search-workflow-a")
	wfA.SetAgentId(agent.GetId())
	wfAResp, _, err := client.DefaultAPI.ApiAmbientApiServerV1WorkflowsPost(ctx).Workflow(*wfA).Execute()
	require.NoError(t, err)

	wfB := openapi.NewWorkflow("search-workflow-b")
	wfB.SetAgentId(agent.GetId())
	wfBResp, _, err := client.DefaultAPI.ApiAmbientApiServerV1WorkflowsPost(ctx).Workflow(*wfB).Execute()
	require.NoError(t, err)

	for i := 0; i < 4; i++ {
		s := openapi.NewSession(fmt.Sprintf("wf-a-session-%d", i))
		s.SetWorkflowId(wfAResp.GetId())
		_, _, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsPost(ctx).Session(*s).Execute()
		require.NoError(t, err)
	}
	s := openapi.NewSession("wf-b-session-0")
	s.SetWorkflowId(wfBResp.GetId())
	_, _, err = client.DefaultAPI.ApiAmbientApiServerV1SessionsPost(ctx).Session(*s).Execute()
	require.NoError(t, err)

	search := fmt.Sprintf("workflow_id = '%s'", wfAResp.GetId())
	list, _, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsGet(ctx).Search(search).Execute()
	require.NoError(t, err)
	assert.Equal(t, int32(4), list.Total)

	search = fmt.Sprintf("workflow_id = '%s'", wfBResp.GetId())
	list, _, err = client.DefaultAPI.ApiAmbientApiServerV1SessionsGet(ctx).Search(search).Execute()
	require.NoError(t, err)
	assert.Equal(t, int32(1), list.Total)
}

func TestSearchWorkflowSkillsByWorkflowId(t *testing.T) {
	helper := test.NewHelper(t)
	client := helper.NewApiClient()
	account := helper.NewRandAccount()
	ctx := helper.NewAuthenticatedContext(account)

	agent, _, err := client.DefaultAPI.ApiAmbientApiServerV1AgentsPost(ctx).Agent(*openapi.NewAgent("ws-search-agent")).Execute()
	require.NoError(t, err)

	wfA := openapi.NewWorkflow("ws-search-wf-a")
	wfA.SetAgentId(agent.GetId())
	wfAResp, _, err := client.DefaultAPI.ApiAmbientApiServerV1WorkflowsPost(ctx).Workflow(*wfA).Execute()
	require.NoError(t, err)

	wfB := openapi.NewWorkflow("ws-search-wf-b")
	wfB.SetAgentId(agent.GetId())
	wfBResp, _, err := client.DefaultAPI.ApiAmbientApiServerV1WorkflowsPost(ctx).Workflow(*wfB).Execute()
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		sk, _, err := client.DefaultAPI.ApiAmbientApiServerV1SkillsPost(ctx).Skill(*openapi.NewSkill(fmt.Sprintf("ws-skill-a-%d", i))).Execute()
		require.NoError(t, err)
		ws := openapi.NewWorkflowSkill(wfAResp.GetId(), sk.GetId(), int32(i+1))
		_, _, err = client.DefaultAPI.ApiAmbientApiServerV1WorkflowSkillsPost(ctx).WorkflowSkill(*ws).Execute()
		require.NoError(t, err)
	}
	sk, _, err := client.DefaultAPI.ApiAmbientApiServerV1SkillsPost(ctx).Skill(*openapi.NewSkill("ws-skill-b-0")).Execute()
	require.NoError(t, err)
	ws := openapi.NewWorkflowSkill(wfBResp.GetId(), sk.GetId(), 1)
	_, _, err = client.DefaultAPI.ApiAmbientApiServerV1WorkflowSkillsPost(ctx).WorkflowSkill(*ws).Execute()
	require.NoError(t, err)

	search := fmt.Sprintf("workflow_id = '%s'", wfAResp.GetId())
	list, _, err := client.DefaultAPI.ApiAmbientApiServerV1WorkflowSkillsGet(ctx).Search(search).Execute()
	require.NoError(t, err)
	assert.Equal(t, int32(3), list.Total)
	for _, item := range list.Items {
		assert.Equal(t, wfAResp.GetId(), item.GetWorkflowId())
	}

	search = fmt.Sprintf("workflow_id = '%s'", wfBResp.GetId())
	list, _, err = client.DefaultAPI.ApiAmbientApiServerV1WorkflowSkillsGet(ctx).Search(search).Execute()
	require.NoError(t, err)
	assert.Equal(t, int32(1), list.Total)
}

func TestSearchWorkflowTasksByWorkflowId(t *testing.T) {
	helper := test.NewHelper(t)
	client := helper.NewApiClient()
	account := helper.NewRandAccount()
	ctx := helper.NewAuthenticatedContext(account)

	agent, _, err := client.DefaultAPI.ApiAmbientApiServerV1AgentsPost(ctx).Agent(*openapi.NewAgent("wt-search-agent")).Execute()
	require.NoError(t, err)

	wf := openapi.NewWorkflow("wt-search-wf")
	wf.SetAgentId(agent.GetId())
	wfResp, _, err := client.DefaultAPI.ApiAmbientApiServerV1WorkflowsPost(ctx).Workflow(*wf).Execute()
	require.NoError(t, err)

	for i := 0; i < 2; i++ {
		tk, _, err := client.DefaultAPI.ApiAmbientApiServerV1TasksPost(ctx).Task(*openapi.NewTask(fmt.Sprintf("wt-task-%d", i))).Execute()
		require.NoError(t, err)
		wt := openapi.NewWorkflowTask(wfResp.GetId(), tk.GetId(), int32(i+1))
		_, _, err = client.DefaultAPI.ApiAmbientApiServerV1WorkflowTasksPost(ctx).WorkflowTask(*wt).Execute()
		require.NoError(t, err)
	}

	search := fmt.Sprintf("workflow_id = '%s'", wfResp.GetId())
	list, _, err := client.DefaultAPI.ApiAmbientApiServerV1WorkflowTasksGet(ctx).Search(search).Execute()
	require.NoError(t, err)
	assert.Equal(t, int32(2), list.Total)
	for _, item := range list.Items {
		assert.Equal(t, wfResp.GetId(), item.GetWorkflowId())
	}
}

func TestSearchWorkflowsByAgentId(t *testing.T) {
	helper := test.NewHelper(t)
	client := helper.NewApiClient()
	account := helper.NewRandAccount()
	ctx := helper.NewAuthenticatedContext(account)

	agentA, _, err := client.DefaultAPI.ApiAmbientApiServerV1AgentsPost(ctx).Agent(*openapi.NewAgent("wf-agent-a")).Execute()
	require.NoError(t, err)
	agentB, _, err := client.DefaultAPI.ApiAmbientApiServerV1AgentsPost(ctx).Agent(*openapi.NewAgent("wf-agent-b")).Execute()
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		wf := openapi.NewWorkflow(fmt.Sprintf("agent-a-wf-%d", i))
		wf.SetAgentId(agentA.GetId())
		_, _, err := client.DefaultAPI.ApiAmbientApiServerV1WorkflowsPost(ctx).Workflow(*wf).Execute()
		require.NoError(t, err)
	}
	wf := openapi.NewWorkflow("agent-b-wf-0")
	wf.SetAgentId(agentB.GetId())
	_, _, err = client.DefaultAPI.ApiAmbientApiServerV1WorkflowsPost(ctx).Workflow(*wf).Execute()
	require.NoError(t, err)

	search := fmt.Sprintf("agent_id = '%s'", agentA.GetId())
	list, _, err := client.DefaultAPI.ApiAmbientApiServerV1WorkflowsGet(ctx).Search(search).Execute()
	require.NoError(t, err)
	assert.Equal(t, int32(3), list.Total)

	search = fmt.Sprintf("agent_id = '%s'", agentB.GetId())
	list, _, err = client.DefaultAPI.ApiAmbientApiServerV1WorkflowsGet(ctx).Search(search).Execute()
	require.NoError(t, err)
	assert.Equal(t, int32(1), list.Total)
}

func TestSearchPermissionsByProjectId(t *testing.T) {
	helper := test.NewHelper(t)
	client := helper.NewApiClient()
	account := helper.NewRandAccount()
	ctx := helper.NewAuthenticatedContext(account)

	projA, _, err := client.DefaultAPI.ApiAmbientApiServerV1ProjectsPost(ctx).Project(*openapi.NewProject("perm-proj-a")).Execute()
	require.NoError(t, err)
	projB, _, err := client.DefaultAPI.ApiAmbientApiServerV1ProjectsPost(ctx).Project(*openapi.NewProject("perm-proj-b")).Execute()
	require.NoError(t, err)

	for i := 0; i < 2; i++ {
		p := openapi.NewPermission("user", fmt.Sprintf("perm-user-%d", i), "edit")
		p.SetProjectId(projA.GetId())
		_, _, err := client.DefaultAPI.ApiAmbientApiServerV1PermissionsPost(ctx).Permission(*p).Execute()
		require.NoError(t, err)
	}
	p := openapi.NewPermission("group", "perm-group-b", "admin")
	p.SetProjectId(projB.GetId())
	_, _, err = client.DefaultAPI.ApiAmbientApiServerV1PermissionsPost(ctx).Permission(*p).Execute()
	require.NoError(t, err)

	search := fmt.Sprintf("project_id = '%s'", projA.GetId())
	list, _, err := client.DefaultAPI.ApiAmbientApiServerV1PermissionsGet(ctx).Search(search).Execute()
	require.NoError(t, err)
	assert.Equal(t, int32(2), list.Total)

	search = fmt.Sprintf("project_id = '%s'", projB.GetId())
	list, _, err = client.DefaultAPI.ApiAmbientApiServerV1PermissionsGet(ctx).Search(search).Execute()
	require.NoError(t, err)
	assert.Equal(t, int32(1), list.Total)
}

func TestSearchRepositoryRefsByProjectId(t *testing.T) {
	helper := test.NewHelper(t)
	client := helper.NewApiClient()
	account := helper.NewRandAccount()
	ctx := helper.NewAuthenticatedContext(account)

	proj, _, err := client.DefaultAPI.ApiAmbientApiServerV1ProjectsPost(ctx).Project(*openapi.NewProject("repo-proj")).Execute()
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		r := openapi.NewRepositoryRef(fmt.Sprintf("repo-%d", i), fmt.Sprintf("https://github.com/test/repo-%d", i))
		r.SetProjectId(proj.GetId())
		_, _, err := client.DefaultAPI.ApiAmbientApiServerV1RepositoryRefsPost(ctx).RepositoryRef(*r).Execute()
		require.NoError(t, err)
	}

	search := fmt.Sprintf("project_id = '%s'", proj.GetId())
	list, _, err := client.DefaultAPI.ApiAmbientApiServerV1RepositoryRefsGet(ctx).Search(search).Execute()
	require.NoError(t, err)
	assert.Equal(t, int32(3), list.Total)
	for _, item := range list.Items {
		assert.Equal(t, proj.GetId(), item.GetProjectId())
	}
}

func TestSearchProjectKeysByProjectId(t *testing.T) {
	helper := test.NewHelper(t)
	client := helper.NewApiClient()
	account := helper.NewRandAccount()
	ctx := helper.NewAuthenticatedContext(account)

	proj, _, err := client.DefaultAPI.ApiAmbientApiServerV1ProjectsPost(ctx).Project(*openapi.NewProject("key-proj")).Execute()
	require.NoError(t, err)

	for i := 0; i < 2; i++ {
		k := openapi.NewProjectKey(fmt.Sprintf("key-%d", i))
		k.SetProjectId(proj.GetId())
		_, _, err := client.DefaultAPI.ApiAmbientApiServerV1ProjectKeysPost(ctx).ProjectKey(*k).Execute()
		require.NoError(t, err)
	}

	search := fmt.Sprintf("project_id = '%s'", proj.GetId())
	list, _, err := client.DefaultAPI.ApiAmbientApiServerV1ProjectKeysGet(ctx).Search(search).Execute()
	require.NoError(t, err)
	assert.Equal(t, int32(2), list.Total)
}

func TestSearchProjectSettingsByProjectId(t *testing.T) {
	helper := test.NewHelper(t)
	client := helper.NewApiClient()
	account := helper.NewRandAccount()
	ctx := helper.NewAuthenticatedContext(account)

	proj, _, err := client.DefaultAPI.ApiAmbientApiServerV1ProjectsPost(ctx).Project(*openapi.NewProject("settings-proj")).Execute()
	require.NoError(t, err)

	ps := openapi.NewProjectSettings(proj.GetId())
	_, _, err = client.DefaultAPI.ApiAmbientApiServerV1ProjectSettingsPost(ctx).ProjectSettings(*ps).Execute()
	require.NoError(t, err)

	search := fmt.Sprintf("project_id = '%s'", proj.GetId())
	list, _, err := client.DefaultAPI.ApiAmbientApiServerV1ProjectSettingsGet(ctx).Search(search).Execute()
	require.NoError(t, err)
	assert.Equal(t, int32(1), list.Total)
	assert.Equal(t, proj.GetId(), list.Items[0].GetProjectId())
}

func TestSearchSessionsByPhase(t *testing.T) {
	helper := test.NewHelper(t)
	client := helper.NewApiClient()
	account := helper.NewRandAccount()
	ctx := helper.NewAuthenticatedContext(account)

	proj, _, err := client.DefaultAPI.ApiAmbientApiServerV1ProjectsPost(ctx).Project(*openapi.NewProject("phase-proj")).Execute()
	require.NoError(t, err)

	s1 := openapi.NewSession("phase-test-1")
	s1.SetProjectId(proj.GetId())
	s1Resp, _, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsPost(ctx).Session(*s1).Execute()
	require.NoError(t, err)
	s2 := openapi.NewSession("phase-test-2")
	s2.SetProjectId(proj.GetId())
	s2Resp, _, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsPost(ctx).Session(*s2).Execute()
	require.NoError(t, err)
	s3 := openapi.NewSession("phase-test-3")
	s3.SetProjectId(proj.GetId())
	_, _, err = client.DefaultAPI.ApiAmbientApiServerV1SessionsPost(ctx).Session(*s3).Execute()
	require.NoError(t, err)

	_, _, err = client.DefaultAPI.ApiAmbientApiServerV1SessionsIdStartPost(ctx, s1Resp.GetId()).Execute()
	require.NoError(t, err)
	_, _, err = client.DefaultAPI.ApiAmbientApiServerV1SessionsIdStartPost(ctx, s2Resp.GetId()).Execute()
	require.NoError(t, err)

	search := fmt.Sprintf("project_id = '%s' and phase = 'Pending'", proj.GetId())
	list, _, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsGet(ctx).Search(search).Execute()
	require.NoError(t, err)
	assert.Equal(t, int32(2), list.Total)
	for _, item := range list.Items {
		assert.Equal(t, "Pending", item.GetPhase())
	}
}

func TestSearchSessionsByNameLike(t *testing.T) {
	helper := test.NewHelper(t)
	client := helper.NewApiClient()
	account := helper.NewRandAccount()
	ctx := helper.NewAuthenticatedContext(account)

	proj, _, err := client.DefaultAPI.ApiAmbientApiServerV1ProjectsPost(ctx).Project(*openapi.NewProject("namelike-proj")).Execute()
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		s := openapi.NewSession(fmt.Sprintf("alpha-session-%d", i))
		s.SetProjectId(proj.GetId())
		_, _, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsPost(ctx).Session(*s).Execute()
		require.NoError(t, err)
	}
	for i := 0; i < 2; i++ {
		s := openapi.NewSession(fmt.Sprintf("beta-session-%d", i))
		s.SetProjectId(proj.GetId())
		_, _, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsPost(ctx).Session(*s).Execute()
		require.NoError(t, err)
	}

	search := fmt.Sprintf("project_id = '%s' and name like 'alpha%%'", proj.GetId())
	list, _, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsGet(ctx).Search(search).Execute()
	require.NoError(t, err)
	assert.Equal(t, int32(3), list.Total)

	search = fmt.Sprintf("project_id = '%s' and name like 'beta%%'", proj.GetId())
	list, _, err = client.DefaultAPI.ApiAmbientApiServerV1SessionsGet(ctx).Search(search).Execute()
	require.NoError(t, err)
	assert.Equal(t, int32(2), list.Total)
}

func TestSearchSessionsCombinedFilters(t *testing.T) {
	helper := test.NewHelper(t)
	client := helper.NewApiClient()
	account := helper.NewRandAccount()
	ctx := helper.NewAuthenticatedContext(account)

	proj, _, err := client.DefaultAPI.ApiAmbientApiServerV1ProjectsPost(ctx).Project(*openapi.NewProject("combo-proj")).Execute()
	require.NoError(t, err)

	for i := 0; i < 3; i++ {
		s := openapi.NewSession(fmt.Sprintf("combo-session-%d", i))
		s.SetProjectId(proj.GetId())
		_, _, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsPost(ctx).Session(*s).Execute()
		require.NoError(t, err)
	}
	other := openapi.NewSession("other-project-session")
	_, _, err = client.DefaultAPI.ApiAmbientApiServerV1SessionsPost(ctx).Session(*other).Execute()
	require.NoError(t, err)

	search := fmt.Sprintf("project_id = '%s' and name like 'combo%%'", proj.GetId())
	list, _, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsGet(ctx).Search(search).Execute()
	require.NoError(t, err)
	assert.Equal(t, int32(3), list.Total)
}

func TestSearchAgentsByProjectId(t *testing.T) {
	helper := test.NewHelper(t)
	client := helper.NewApiClient()
	account := helper.NewRandAccount()
	ctx := helper.NewAuthenticatedContext(account)

	proj, _, err := client.DefaultAPI.ApiAmbientApiServerV1ProjectsPost(ctx).Project(*openapi.NewProject("agent-proj")).Execute()
	require.NoError(t, err)

	for i := 0; i < 2; i++ {
		a := openapi.NewAgent(fmt.Sprintf("proj-agent-%d", i))
		a.SetProjectId(proj.GetId())
		_, _, err := client.DefaultAPI.ApiAmbientApiServerV1AgentsPost(ctx).Agent(*a).Execute()
		require.NoError(t, err)
	}
	_, _, err = client.DefaultAPI.ApiAmbientApiServerV1AgentsPost(ctx).Agent(*openapi.NewAgent("orphan-agent")).Execute()
	require.NoError(t, err)

	search := fmt.Sprintf("project_id = '%s'", proj.GetId())
	list, _, err := client.DefaultAPI.ApiAmbientApiServerV1AgentsGet(ctx).Search(search).Execute()
	require.NoError(t, err)
	assert.Equal(t, int32(2), list.Total)
}

func TestSearchSkillsByProjectId(t *testing.T) {
	helper := test.NewHelper(t)
	client := helper.NewApiClient()
	account := helper.NewRandAccount()
	ctx := helper.NewAuthenticatedContext(account)

	proj, _, err := client.DefaultAPI.ApiAmbientApiServerV1ProjectsPost(ctx).Project(*openapi.NewProject("skill-proj")).Execute()
	require.NoError(t, err)

	for i := 0; i < 2; i++ {
		s := openapi.NewSkill(fmt.Sprintf("proj-skill-%d", i))
		s.SetProjectId(proj.GetId())
		_, _, err := client.DefaultAPI.ApiAmbientApiServerV1SkillsPost(ctx).Skill(*s).Execute()
		require.NoError(t, err)
	}

	search := fmt.Sprintf("project_id = '%s'", proj.GetId())
	list, _, err := client.DefaultAPI.ApiAmbientApiServerV1SkillsGet(ctx).Search(search).Execute()
	require.NoError(t, err)
	assert.Equal(t, int32(2), list.Total)
}

func TestSearchTasksByProjectId(t *testing.T) {
	helper := test.NewHelper(t)
	client := helper.NewApiClient()
	account := helper.NewRandAccount()
	ctx := helper.NewAuthenticatedContext(account)

	proj, _, err := client.DefaultAPI.ApiAmbientApiServerV1ProjectsPost(ctx).Project(*openapi.NewProject("task-proj")).Execute()
	require.NoError(t, err)

	for i := 0; i < 2; i++ {
		tk := openapi.NewTask(fmt.Sprintf("proj-task-%d", i))
		tk.SetProjectId(proj.GetId())
		_, _, err := client.DefaultAPI.ApiAmbientApiServerV1TasksPost(ctx).Task(*tk).Execute()
		require.NoError(t, err)
	}

	search := fmt.Sprintf("project_id = '%s'", proj.GetId())
	list, _, err := client.DefaultAPI.ApiAmbientApiServerV1TasksGet(ctx).Search(search).Execute()
	require.NoError(t, err)
	assert.Equal(t, int32(2), list.Total)
}

func TestSearchWorkflowsByProjectId(t *testing.T) {
	helper := test.NewHelper(t)
	client := helper.NewApiClient()
	account := helper.NewRandAccount()
	ctx := helper.NewAuthenticatedContext(account)

	proj, _, err := client.DefaultAPI.ApiAmbientApiServerV1ProjectsPost(ctx).Project(*openapi.NewProject("wf-proj")).Execute()
	require.NoError(t, err)

	for i := 0; i < 2; i++ {
		wf := openapi.NewWorkflow(fmt.Sprintf("proj-wf-%d", i))
		wf.SetProjectId(proj.GetId())
		_, _, err := client.DefaultAPI.ApiAmbientApiServerV1WorkflowsPost(ctx).Workflow(*wf).Execute()
		require.NoError(t, err)
	}

	search := fmt.Sprintf("project_id = '%s'", proj.GetId())
	list, _, err := client.DefaultAPI.ApiAmbientApiServerV1WorkflowsGet(ctx).Search(search).Execute()
	require.NoError(t, err)
	assert.Equal(t, int32(2), list.Total)
}
