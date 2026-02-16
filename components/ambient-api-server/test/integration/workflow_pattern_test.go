package integration

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ambient/platform/components/ambient-api-server/pkg/api/openapi"
	"github.com/ambient/platform/components/ambient-api-server/test"
)

func TestWorkflowPattern_AsAgentWithSkillsDoTasks(t *testing.T) {
	helper := test.NewHelper(t)
	client := helper.NewApiClient()

	// Create test account for authentication
	account := helper.NewRandAccount()
	ctx := helper.NewAuthenticatedContext(account)

	// Test data for the "AS agent WITH skills DO tasks" pattern
	testData := setupWorkflowTestData(t, client, ctx)

	// Test 1: Create Agent
	t.Run("CreateAgent", func(t *testing.T) {
		agent := testData.agent
		
		resp, httpResp, err := client.DefaultAPI.ApiAmbientApiServerV1AgentsPost(ctx).Agent(agent).Execute()
		require.NoError(t, err, "Failed to create agent")
		require.NotNil(t, httpResp, "HTTP response should not be nil")
		assert.Equal(t, 201, httpResp.StatusCode, "Should return 201 Created")

		require.NotNil(t, resp, "Response should not be nil")
		assert.Equal(t, agent.GetName(), resp.GetName(), "Agent name should match")
		assert.NotEmpty(t, resp.GetId(), "Agent should have an ID")
		
		testData.createdAgentId = resp.GetId()
	})

	// Test 2: Create Skills
	t.Run("CreateSkills", func(t *testing.T) {
		for i, skill := range testData.skills {
			resp, httpResp, err := client.DefaultAPI.ApiAmbientApiServerV1SkillsPost(ctx).Skill(skill).Execute()
			require.NoError(t, err, "Failed to create skill %s", skill.GetName())
			require.NotNil(t, httpResp, "HTTP response should not be nil")
			assert.Equal(t, 201, httpResp.StatusCode, "Should return 201 Created")

			require.NotNil(t, resp, "Response should not be nil")
			assert.Equal(t, skill.GetName(), resp.GetName(), "Skill name should match")
			assert.NotEmpty(t, resp.GetId(), "Skill should have an ID")
			
			testData.createdSkillIds[i] = resp.GetId()
		}
	})

	// Test 3: Create Tasks
	t.Run("CreateTasks", func(t *testing.T) {
		for i, task := range testData.tasks {
			resp, httpResp, err := client.DefaultAPI.ApiAmbientApiServerV1TasksPost(ctx).Task(task).Execute()
			require.NoError(t, err, "Failed to create task %s", task.GetName())
			require.NotNil(t, httpResp, "HTTP response should not be nil")
			assert.Equal(t, 201, httpResp.StatusCode, "Should return 201 Created")

			require.NotNil(t, resp, "Response should not be nil")
			assert.Equal(t, task.GetName(), resp.GetName(), "Task name should match")
			assert.NotEmpty(t, resp.GetId(), "Task should have an ID")
			
			testData.createdTaskIds[i] = resp.GetId()
		}
	})

	// Test 4: Create Workflow (AS agent WITH skills DO tasks)
	t.Run("CreateWorkflow", func(t *testing.T) {
		workflow := testData.workflow
		
		resp, httpResp, err := client.DefaultAPI.ApiAmbientApiServerV1WorkflowsPost(ctx).Workflow(workflow).Execute()
		require.NoError(t, err, "Failed to create workflow")
		require.NotNil(t, httpResp, "HTTP response should not be nil")
		assert.Equal(t, 201, httpResp.StatusCode, "Should return 201 Created")

		require.NotNil(t, resp, "Response should not be nil")
		assert.Equal(t, workflow.GetName(), resp.GetName(), "Workflow name should match")
		assert.NotEmpty(t, resp.GetId(), "Workflow should have an ID")
		
		testData.createdWorkflowId = resp.GetId()
	})

	// Test 5: Associate Skills to Workflow (WITH skills part)
	t.Run("AssociateSkillsToWorkflow", func(t *testing.T) {
		for i, skillId := range testData.createdSkillIds {
			workflowSkill := openapi.NewWorkflowSkill(testData.createdWorkflowId, skillId, int32(i+1))
			
			resp, httpResp, err := client.DefaultAPI.ApiAmbientApiServerV1WorkflowSkillsPost(ctx).WorkflowSkill(*workflowSkill).Execute()
			require.NoError(t, err, "Failed to associate skill to workflow")
			require.NotNil(t, httpResp, "HTTP response should not be nil")
			assert.Equal(t, 201, httpResp.StatusCode, "Should return 201 Created")

			require.NotNil(t, resp, "Response should not be nil")
			assert.Equal(t, testData.createdWorkflowId, resp.GetWorkflowId(), "Workflow ID should match")
			assert.Equal(t, skillId, resp.GetSkillId(), "Skill ID should match")
			assert.Equal(t, int32(i+1), resp.GetPosition(), "Position should match")
		}
	})

	// Test 6: Associate Tasks to Workflow (DO tasks part)
	t.Run("AssociateTasksToWorkflow", func(t *testing.T) {
		for i, taskId := range testData.createdTaskIds {
			workflowTask := openapi.NewWorkflowTask(testData.createdWorkflowId, taskId, int32(i+1))
			
			resp, httpResp, err := client.DefaultAPI.ApiAmbientApiServerV1WorkflowTasksPost(ctx).WorkflowTask(*workflowTask).Execute()
			require.NoError(t, err, "Failed to associate task to workflow")
			require.NotNil(t, httpResp, "HTTP response should not be nil")
			assert.Equal(t, 201, httpResp.StatusCode, "Should return 201 Created")

			require.NotNil(t, resp, "Response should not be nil")
			assert.Equal(t, testData.createdWorkflowId, resp.GetWorkflowId(), "Workflow ID should match")
			assert.Equal(t, taskId, resp.GetTaskId(), "Task ID should match")
			assert.Equal(t, int32(i+1), resp.GetPosition(), "Position should match")
		}
	})

	// Test 7: Create Session that instantiates the workflow
	t.Run("CreateSessionWithWorkflow", func(t *testing.T) {
		session := testData.session
		session.SetWorkflowId(testData.createdWorkflowId)
		
		resp, httpResp, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsPost(ctx).Session(session).Execute()
		require.NoError(t, err, "Failed to create session")
		require.NotNil(t, httpResp, "HTTP response should not be nil")
		assert.Equal(t, 201, httpResp.StatusCode, "Should return 201 Created")

		require.NotNil(t, resp, "Response should not be nil")
		assert.Equal(t, session.GetName(), resp.GetName(), "Session name should match")
		assert.Equal(t, testData.createdWorkflowId, resp.GetWorkflowId(), "Workflow ID should match")
		assert.NotEmpty(t, resp.GetId(), "Session should have an ID")
		
		testData.createdSessionId = resp.GetId()
	})

	// Test 8: Verify complete workflow pattern structure
	t.Run("VerifyWorkflowPattern", func(t *testing.T) {
		// Get workflow with all associations
		workflow, httpResp, err := client.DefaultAPI.ApiAmbientApiServerV1WorkflowsIdGet(ctx, testData.createdWorkflowId).Execute()
		require.NoError(t, err, "Failed to get workflow")
		require.NotNil(t, httpResp, "HTTP response should not be nil")
		assert.Equal(t, 200, httpResp.StatusCode, "Should return 200 OK")

		// Verify workflow exists and has correct structure
		assert.Equal(t, "Code Analysis Workflow", workflow.GetName())
		assert.Equal(t, "https://github.com/ambient/workflows/code-analysis", workflow.GetRepoUrl())
		
		// Get workflow skills and verify order
		skillsResp, _, err := client.DefaultAPI.ApiAmbientApiServerV1WorkflowSkillsGet(ctx).Execute()
		require.NoError(t, err, "Failed to get workflow skills")
		
		workflowSkills := filterWorkflowSkillsByWorkflowId(skillsResp.GetItems(), testData.createdWorkflowId)
		assert.Len(t, workflowSkills, 3, "Should have 3 skills associated")
		
		// Verify skills are ordered correctly
		sortWorkflowSkillsByPosition(workflowSkills)
		expectedSkillNames := []string{"API Design", "Code Analysis", "Testing"}
		for i, ws := range workflowSkills {
			// Get skill details to verify name
			skill, _, err := client.DefaultAPI.ApiAmbientApiServerV1SkillsIdGet(ctx, ws.GetSkillId()).Execute()
			require.NoError(t, err, "Failed to get skill details")
			assert.Equal(t, expectedSkillNames[i], skill.GetName(), "Skill should be in correct order")
		}
		
		// Get workflow tasks and verify order
		tasksResp, _, err := client.DefaultAPI.ApiAmbientApiServerV1WorkflowTasksGet(ctx).Execute()
		require.NoError(t, err, "Failed to get workflow tasks")
		
		workflowTasks := filterWorkflowTasksByWorkflowId(tasksResp.GetItems(), testData.createdWorkflowId)
		assert.Len(t, workflowTasks, 2, "Should have 2 tasks associated")
		
		// Verify tasks are ordered correctly
		sortWorkflowTasksByPosition(workflowTasks)
		expectedTaskNames := []string{"Analyze Code Quality", "Generate Report"}
		for i, wt := range workflowTasks {
			// Get task details to verify name
			task, _, err := client.DefaultAPI.ApiAmbientApiServerV1TasksIdGet(ctx, wt.GetTaskId()).Execute()
			require.NoError(t, err, "Failed to get task details")
			assert.Equal(t, expectedTaskNames[i], task.GetName(), "Task should be in correct order")
		}
	})

	// Test 9: Test pagination and search functionality
	t.Run("TestPaginationAndSearch", func(t *testing.T) {
		// Test agents search
		agentsResp, _, err := client.DefaultAPI.ApiAmbientApiServerV1AgentsGet(ctx).
			Search("name like 'Claude%'").
			Page(1).
			Size(10).
			Execute()
		require.NoError(t, err, "Failed to search agents")
		assert.GreaterOrEqual(t, len(agentsResp.GetItems()), 1, "Should find at least one agent")
		
		// Test skills search
		skillsResp, _, err := client.DefaultAPI.ApiAmbientApiServerV1SkillsGet(ctx).
			Search("name like 'API%'").
			Execute()
		require.NoError(t, err, "Failed to search skills")
		assert.GreaterOrEqual(t, len(skillsResp.GetItems()), 1, "Should find at least one skill")
	})

	// Test 10: Test error handling
	t.Run("TestErrorHandling", func(t *testing.T) {
		// Try to create duplicate agent (should fail)
		duplicateAgent := testData.agent
		duplicateAgent.SetName("Claude Code Assistant") // Same name
		
		_, httpResp, err := client.DefaultAPI.ApiAmbientApiServerV1AgentsPost(ctx).Agent(duplicateAgent).Execute()
		if err != nil {
			// Should get a conflict error or validation error
			assert.NotNil(t, httpResp, "Should have HTTP response even on error")
			assert.True(t, httpResp.StatusCode >= 400, "Should return error status code")
		}
		
		// Try to get non-existent resource
		_, httpResp, err = client.DefaultAPI.ApiAmbientApiServerV1AgentsIdGet(ctx, "non-existent-id").Execute()
		require.Error(t, err, "Should get error for non-existent resource")
		assert.Equal(t, 404, httpResp.StatusCode, "Should return 404 Not Found")
	})
}

// Test data structure to manage related entities
type workflowTestData struct {
	agent     openapi.Agent
	skills    []openapi.Skill
	tasks     []openapi.Task
	workflow  openapi.Workflow
	session   openapi.Session
	
	// Created IDs for cleanup and verification
	createdAgentId    string
	createdSkillIds   []string
	createdTaskIds    []string
	createdWorkflowId string
	createdSessionId  string
}

func setupWorkflowTestData(t *testing.T, client *openapi.APIClient, ctx context.Context) *workflowTestData {
	// Create test user first
	user := openapi.NewUser("test-user", "Test User")
	
	userResp, _, err := client.DefaultAPI.ApiAmbientApiServerV1UsersPost(ctx).User(*user).Execute()
	require.NoError(t, err, "Failed to create test user")
	createdUserId := userResp.GetId()

	// Agent: Claude Code Assistant
	agent := openapi.NewAgent("Claude Code Assistant")
	agent.SetRepoUrl("https://github.com/ambient/agents/claude-code-assistant")
	agent.SetPrompt("AI assistant specialized in code analysis and software development tasks")

	// Skills: API Design, Code Analysis, Testing
	skills := []openapi.Skill{
		*openapi.NewSkill("API Design"),
		*openapi.NewSkill("Code Analysis"),
		*openapi.NewSkill("Testing"),
	}
	
	skills[0].SetRepoUrl("https://github.com/ambient/skills/api-design")
	skills[0].SetPrompt("Expertise in REST API design patterns and best practices")
	
	skills[1].SetRepoUrl("https://github.com/ambient/skills/code-analysis")
	skills[1].SetPrompt("Static code analysis, complexity metrics, and code quality assessment")
	
	skills[2].SetRepoUrl("https://github.com/ambient/skills/testing")
	skills[2].SetPrompt("Test strategy, test case generation, and quality assurance")

	// Tasks: Analyze Code Quality, Generate Report
	tasks := []openapi.Task{
		*openapi.NewTask("Analyze Code Quality"),
		*openapi.NewTask("Generate Report"),
	}
	
	tasks[0].SetRepoUrl("https://github.com/ambient/tasks/code-quality-analysis")
	tasks[0].SetPrompt("Perform comprehensive code quality analysis including complexity, maintainability, and best practices")
	
	tasks[1].SetRepoUrl("https://github.com/ambient/tasks/report-generation")
	tasks[1].SetPrompt("Generate detailed reports with findings, recommendations, and actionable insights")

	// Workflow: Code Analysis Workflow
	workflow := openapi.NewWorkflow("Code Analysis Workflow")
	workflow.SetRepoUrl("https://github.com/ambient/workflows/code-analysis")
	workflow.SetPrompt("Comprehensive code analysis workflow combining API design, code analysis, and testing skills")

	// Session: Code Review Session
	session := openapi.NewSession("Code Review Session")
	session.SetRepoUrl("https://github.com/test-project/sample-code")
	session.SetPrompt("Please analyze this codebase for quality issues and provide improvement recommendations")
	session.SetCreatedByUserId(createdUserId)

	return &workflowTestData{
		agent:           *agent,
		skills:          skills,
		tasks:           tasks,
		workflow:        *workflow,
		session:         *session,
		createdSkillIds: make([]string, len(skills)),
		createdTaskIds:  make([]string, len(tasks)),
	}
}

// Helper functions for filtering and sorting
func filterWorkflowSkillsByWorkflowId(items []openapi.WorkflowSkill, workflowId string) []openapi.WorkflowSkill {
	var filtered []openapi.WorkflowSkill
	for _, item := range items {
		if item.GetWorkflowId() == workflowId {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func filterWorkflowTasksByWorkflowId(items []openapi.WorkflowTask, workflowId string) []openapi.WorkflowTask {
	var filtered []openapi.WorkflowTask
	for _, item := range items {
		if item.GetWorkflowId() == workflowId {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func sortWorkflowSkillsByPosition(items []openapi.WorkflowSkill) {
	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			if items[i].GetPosition() > items[j].GetPosition() {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}

func sortWorkflowTasksByPosition(items []openapi.WorkflowTask) {
	for i := 0; i < len(items)-1; i++ {
		for j := i + 1; j < len(items); j++ {
			if items[i].GetPosition() > items[j].GetPosition() {
				items[i], items[j] = items[j], items[i]
			}
		}
	}
}