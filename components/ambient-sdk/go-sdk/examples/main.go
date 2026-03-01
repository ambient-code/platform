package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/client"
	"github.com/ambient-code/platform/components/ambient-sdk/go-sdk/types"
)

func main() {
	fmt.Println("Ambient Platform SDK — End-to-End Example")
	fmt.Println("==========================================")
	fmt.Println()

	projectName := "sdk-demo"

	c, err := client.NewClientFromEnv(projectName, client.WithTimeout(120 * time.Second))
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	ctx := context.Background()

	runFullLifecycle(ctx, c, projectName)
}

func runFullLifecycle(ctx context.Context, c *client.Client, projectName string) {
	fmt.Println("Step 1: Create Workspace (Project)")
	fmt.Println("-----------------------------------")

	project, err := types.NewProjectBuilder().
		Name(projectName).
		DisplayName("SDK Demo Workspace").
		Description("Created programmatically via Go SDK").
		Build()
	if err != nil {
		log.Fatalf("Failed to build project: %v", err)
	}

	createdProject, err := c.Projects().Create(ctx, project)
	if err != nil {
		var apiErr *types.APIError
		if ok := asAPIError(err, &apiErr); ok && apiErr.StatusCode == http.StatusConflict {
			fmt.Printf("  Project %q already exists, reusing it\n", projectName)
			createdProject, err = c.Projects().Get(ctx, projectName)
			if err != nil {
				log.Fatalf("Failed to get existing project: %v", err)
			}
		} else {
			log.Fatalf("Failed to create project: %v", err)
		}
	} else {
		fmt.Printf("  Created project: %s (id=%s)\n", createdProject.Name, createdProject.ID)
	}
	fmt.Println()

	fmt.Println("Step 2: Create Session with Repository Context")
	fmt.Println("-----------------------------------------------")

	reposJSON, _ := json.Marshal([]map[string]interface{}{
		{
			"url":      "https://github.com/ambient-code/platform",
			"branch":   "main",
			"autoPush": false,
		},
	})

	session, err := types.NewSessionBuilder().
		Name("sdk-demo-session").
		Prompt("Analyze the repository structure and summarize the key components").
		Timeout(300).
		LlmModel("claude-sonnet-4-20250514").
		LlmTemperature(0.7).
		LlmMaxTokens(4096).
		Repos(string(reposJSON)).
		Build()
	if err != nil {
		log.Fatalf("Failed to build session: %v", err)
	}

	createdSession, err := c.Sessions().Create(ctx, session)
	if err != nil {
		log.Fatalf("Failed to create session: %v", err)
	}

	fmt.Printf("  Created session: %s\n", createdSession.Name)
	fmt.Printf("  Session ID:      %s\n", createdSession.ID)
	fmt.Printf("  Phase:           %s\n", phaseOrDefault(createdSession.Phase))
	fmt.Printf("  Model:           %s\n", createdSession.LlmModel)
	fmt.Printf("  Repos:           %s\n", createdSession.Repos)
	fmt.Println()

	fmt.Println("Step 3: Start Session")
	fmt.Println("---------------------")

	startedSession, err := c.Sessions().Start(ctx, createdSession.ID)
	if err != nil {
		log.Fatalf("Failed to start session: %v", err)
	}
	fmt.Printf("  Start requested. Phase: %s\n", phaseOrDefault(startedSession.Phase))
	fmt.Println()

	fmt.Println("Step 4: Wait for Session to Reach Running Phase")
	fmt.Println("------------------------------------------------")

	runningSession, err := waitForPhase(ctx, c, createdSession.ID, "Running", 5*time.Minute)
	if err != nil {
		fmt.Printf("  Warning: %v\n", err)
		fmt.Println("  (The session may still be starting — the operator creates a runner pod)")
		fmt.Println("  Continuing with demonstration...")
	} else {
		fmt.Printf("  Session is Running! Phase: %s\n", runningSession.Phase)
		if runningSession.StartTime != nil {
			fmt.Printf("  Started at: %s\n", runningSession.StartTime.Format(time.RFC3339))
		}
	}
	fmt.Println()

	fmt.Println("Step 5: Send a Message via AG-UI")
	fmt.Println("--------------------------------")

	apiURL := os.Getenv("AMBIENT_API_URL")
	token := os.Getenv("AMBIENT_TOKEN")

	aguiBaseURL := deriveAGUIBaseURL(apiURL, projectName, createdSession.KubeCrName, createdSession.ID)

	fmt.Printf("  AG-UI endpoint: %s\n", aguiBaseURL)
	fmt.Println("  Sending user message...")

	reply, err := sendMessageAndWaitForReply(ctx, aguiBaseURL, token, createdSession.ID, "Describe the backend components in this repository")
	if err != nil {
		fmt.Printf("  AG-UI messaging not available: %v\n", err)
		fmt.Println("  (This requires the full stack: operator + runner pod running in the cluster)")
		fmt.Println("  The session is created and visible in the UI — you can chat with it there.")
	} else {
		fmt.Println("  Assistant reply (first 500 chars):")
		fmt.Println("  " + truncate(reply, 500))
	}
	fmt.Println()

	fmt.Println("Step 6: Verify Session in List")
	fmt.Println("------------------------------")

	listOpts := types.NewListOptions().Size(10).Build()
	sessionList, err := c.Sessions().List(ctx, listOpts)
	if err != nil {
		log.Fatalf("Failed to list sessions: %v", err)
	}

	fmt.Printf("  Total sessions in project: %d\n", sessionList.Total)
	for i, s := range sessionList.Items {
		if i >= 5 {
			fmt.Printf("  ... and %d more\n", len(sessionList.Items)-5)
			break
		}
		fmt.Printf("  %d. %s (phase=%s, model=%s)\n", i+1, s.Name, phaseOrDefault(s.Phase), s.LlmModel)
	}
	fmt.Println()

	fmt.Println("Step 7: Stop Session")
	fmt.Println("--------------------")

	stoppedSession, err := c.Sessions().Stop(ctx, createdSession.ID)
	if err != nil {
		fmt.Printf("  Warning: could not stop session: %v\n", err)
	} else {
		fmt.Printf("  Stop requested. Phase: %s\n", phaseOrDefault(stoppedSession.Phase))
	}
	fmt.Println()

	fmt.Println("Complete!")
	fmt.Println("=========")
	fmt.Printf("Project:  %s\n", projectName)
	fmt.Printf("Session:  %s (id=%s)\n", createdSession.Name, createdSession.ID)
	fmt.Println("Open the Ambient UI to see the workspace and session.")
}

func waitForPhase(ctx context.Context, c *client.Client, sessionID, targetPhase string, timeout time.Duration) (*types.Session, error) {
	deadline := time.Now().Add(timeout)
	poll := 3 * time.Second

	for time.Now().Before(deadline) {
		session, err := c.Sessions().Get(ctx, sessionID)
		if err != nil {
			return nil, fmt.Errorf("failed to get session: %w", err)
		}

		currentPhase := strings.ToLower(session.Phase)
		target := strings.ToLower(targetPhase)

		if currentPhase == target {
			return session, nil
		}

		if currentPhase == "failed" || currentPhase == "completed" {
			return session, fmt.Errorf("session reached terminal phase %q before %q", session.Phase, targetPhase)
		}

		fmt.Printf("  Phase: %s (waiting for %s...)\n", phaseOrDefault(session.Phase), targetPhase)

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(poll):
		}
	}

	return nil, fmt.Errorf("timed out waiting for phase %q after %v", targetPhase, timeout)
}

func sendMessageAndWaitForReply(ctx context.Context, aguiBaseURL, token, sessionID, message string) (string, error) {
	runPayload := map[string]interface{}{
		"threadId": sessionID,
		"messages": []map[string]interface{}{
			{
				"id":      fmt.Sprintf("msg-%d", time.Now().UnixNano()),
				"role":    "user",
				"content": message,
			},
		},
	}

	body, _ := json.Marshal(runPayload)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, aguiBaseURL+"/run", strings.NewReader(string(body)))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	httpClient := &http.Client{Timeout: 30 * time.Second}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("run endpoint returned %d", resp.StatusCode)
	}

	var runResult map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&runResult); err != nil {
		return "", fmt.Errorf("decode run response: %w", err)
	}

	streamURL := aguiBaseURL + "/events"
	if runID, ok := runResult["runId"].(string); ok {
		streamURL += "?runId=" + runID
	}

	return consumeEventStream(ctx, streamURL, token, 2*time.Minute)
}

func consumeEventStream(ctx context.Context, streamURL, token string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, streamURL, nil)
	if err != nil {
		return "", fmt.Errorf("create SSE request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	httpClient := &http.Client{Timeout: 0}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("connect to event stream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("event stream returned %d", resp.StatusCode)
	}

	var reply strings.Builder
	scanner := bufio.NewScanner(resp.Body)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		jsonData := strings.TrimPrefix(line, "data: ")
		var event map[string]interface{}
		if err := json.Unmarshal([]byte(jsonData), &event); err != nil {
			continue
		}

		eventType, _ := event["type"].(string)

		switch eventType {
		case "TEXT_MESSAGE_CONTENT":
			if delta, ok := event["delta"].(string); ok {
				reply.WriteString(delta)
			}
		case "RUN_FINISHED", "RUN_ERROR":
			return reply.String(), nil
		}
	}

	if reply.Len() > 0 {
		return reply.String(), nil
	}
	return "", fmt.Errorf("event stream ended without a complete reply")
}

func deriveAGUIBaseURL(apiURL, projectName, kubeCRName, sessionID string) string {
	sessionName := kubeCRName
	if sessionName == "" {
		sessionName = sessionID
	}

	baseURL := strings.TrimRight(apiURL, "/")
	if strings.Contains(baseURL, "ambient-api-server") {
		baseURL = strings.TrimSuffix(baseURL, "/api/ambient/v1")
	}

	return fmt.Sprintf("%s/api/projects/%s/agentic-sessions/%s/agui", baseURL, projectName, sessionName)
}

func phaseOrDefault(phase string) string {
	if phase == "" {
		return "Pending"
	}
	return phase
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func asAPIError(err error, target **types.APIError) bool {
	if err == nil {
		return false
	}
	if e, ok := err.(*types.APIError); ok {
		*target = e
		return true
	}
	return false
}
