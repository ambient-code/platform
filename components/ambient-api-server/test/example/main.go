package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	openapi "github.com/ambient/platform/components/ambient-api-server/pkg/api/openapi"
)

func main() {
	serverURL := flag.String("server", "http://localhost:8000", "API server base URL")
	debug := flag.Bool("debug", false, "enable HTTP debug logging")
	flag.Parse()

	cfg := openapi.NewConfiguration()
	cfg.Servers = openapi.ServerConfigurations{
		{URL: *serverURL},
	}
	cfg.Debug = *debug

	client := openapi.NewAPIClient(cfg)
	ctx := context.Background()

	fmt.Println("=== Create Agent ===")
	agent := openapi.NewAgent("test-agent")
	agent.SetPrompt("You are a helpful coding assistant.")
	createdAgent, _, err := client.DefaultAPI.ApiAmbientApiServerV1AgentsPost(ctx).Agent(*agent).Execute()
	if err != nil {
		log.Fatalf("create agent: %v", err)
	}
	prettyPrint(createdAgent)

	fmt.Println("\n=== List Agents ===")
	agentList, _, err := client.DefaultAPI.ApiAmbientApiServerV1AgentsGet(ctx).Execute()
	if err != nil {
		log.Fatalf("list agents: %v", err)
	}
	prettyPrint(agentList)

	fmt.Println("\n=== Create Workflow ===")
	workflow := openapi.NewWorkflow("test-workflow")
	workflow.SetPrompt("Review and refactor code.")
	workflow.SetAgentId(createdAgent.GetId())
	createdWorkflow, _, err := client.DefaultAPI.ApiAmbientApiServerV1WorkflowsPost(ctx).Workflow(*workflow).Execute()
	if err != nil {
		log.Fatalf("create workflow: %v", err)
	}
	prettyPrint(createdWorkflow)

	fmt.Println("\n=== List Workflows ===")
	workflowList, _, err := client.DefaultAPI.ApiAmbientApiServerV1WorkflowsGet(ctx).Execute()
	if err != nil {
		log.Fatalf("list workflows: %v", err)
	}
	prettyPrint(workflowList)

	fmt.Println("\n=== Create User ===")
	user := openapi.NewUser("jdoe", "Jane Doe")
	createdUser, _, err := client.DefaultAPI.ApiAmbientApiServerV1UsersPost(ctx).User(*user).Execute()
	if err != nil {
		log.Fatalf("create user: %v", err)
	}
	prettyPrint(createdUser)

	fmt.Println("\n=== List Users ===")
	userList, _, err := client.DefaultAPI.ApiAmbientApiServerV1UsersGet(ctx).Execute()
	if err != nil {
		log.Fatalf("list users: %v", err)
	}
	prettyPrint(userList)

	fmt.Println("\n=== Create Session ===")
	session := openapi.NewSession("test-session")
	session.SetPrompt("Implement a REST endpoint for health checks.")
	session.SetWorkflowId(createdWorkflow.GetId())
	session.SetCreatedByUserId(createdUser.GetId())
	createdSession, _, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsPost(ctx).Session(*session).Execute()
	if err != nil {
		log.Fatalf("create session: %v", err)
	}
	prettyPrint(createdSession)

	fmt.Println("\n=== List Sessions ===")
	sessionList, _, err := client.DefaultAPI.ApiAmbientApiServerV1SessionsGet(ctx).Execute()
	if err != nil {
		log.Fatalf("list sessions: %v", err)
	}
	prettyPrint(sessionList)

	fmt.Println("\n=== Create Skill ===")
	skill := openapi.NewSkill("code-review")
	skill.SetPrompt("Perform thorough code review with security focus.")
	createdSkill, _, err := client.DefaultAPI.ApiAmbientApiServerV1SkillsPost(ctx).Skill(*skill).Execute()
	if err != nil {
		log.Fatalf("create skill: %v", err)
	}
	prettyPrint(createdSkill)

	fmt.Println("\n=== List Skills ===")
	skillList, _, err := client.DefaultAPI.ApiAmbientApiServerV1SkillsGet(ctx).Execute()
	if err != nil {
		log.Fatalf("list skills: %v", err)
	}
	prettyPrint(skillList)

	fmt.Println("\n=== Create Task ===")
	task := openapi.NewTask("refactor-handlers")
	task.SetPrompt("Break large handler file into smaller modules.")
	createdTask, _, err := client.DefaultAPI.ApiAmbientApiServerV1TasksPost(ctx).Task(*task).Execute()
	if err != nil {
		log.Fatalf("create task: %v", err)
	}
	prettyPrint(createdTask)

	fmt.Println("\n=== List Tasks ===")
	taskList, _, err := client.DefaultAPI.ApiAmbientApiServerV1TasksGet(ctx).Execute()
	if err != nil {
		log.Fatalf("list tasks: %v", err)
	}
	prettyPrint(taskList)

	fmt.Println("\nAll resources created and listed successfully.")
}

func prettyPrint(v any) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal error: %v\n", err)
		return
	}
	fmt.Println(string(data))
}
