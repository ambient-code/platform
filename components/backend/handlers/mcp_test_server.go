package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/kubernetes"
)

// mcpTestScript is an inline Python script that runs inside the test Pod.
// It starts the MCP server process, performs the JSON-RPC initialize handshake
// over stdio with Content-Length framing, and outputs a single JSON line.
const mcpTestScript = `
import subprocess, json, sys, os, time

def main():
    cmd = os.environ.get("MCP_TEST_COMMAND", "")
    args_json = os.environ.get("MCP_TEST_ARGS", "[]")
    env_json = os.environ.get("MCP_TEST_ENV", "{}")
    if not cmd:
        print(json.dumps({"success": False, "error": "MCP_TEST_COMMAND not set"}))
        sys.exit(1)

    try:
        args = json.loads(args_json)
    except Exception:
        args = []
    try:
        extra_env = json.loads(env_json)
    except Exception:
        extra_env = {}

    proc_env = os.environ.copy()
    proc_env.update(extra_env)

    try:
        proc = subprocess.Popen(
            [cmd] + args,
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            env=proc_env,
        )
    except Exception as e:
        print(json.dumps({"success": False, "error": f"Failed to start process: {e}"}))
        sys.exit(1)

    init_req = json.dumps({
        "jsonrpc": "2.0",
        "id": 1,
        "method": "initialize",
        "params": {
            "protocolVersion": "2024-11-05",
            "capabilities": {},
            "clientInfo": {"name": "mcp-test", "version": "1.0.0"}
        }
    })
    msg = f"Content-Length: {len(init_req)}\r\n\r\n{init_req}"

    try:
        proc.stdin.write(msg.encode())
        proc.stdin.flush()
    except Exception as e:
        proc.kill()
        print(json.dumps({"success": False, "error": f"Failed to send initialize: {e}"}))
        sys.exit(1)

    deadline = time.time() + 30
    header = b""
    try:
        while time.time() < deadline:
            b = proc.stdout.read(1)
            if not b:
                break
            header += b
            if header.endswith(b"\r\n\r\n"):
                break

        content_length = 0
        for line in header.decode().split("\r\n"):
            if line.lower().startswith("content-length:"):
                content_length = int(line.split(":", 1)[1].strip())

        if content_length == 0:
            proc.kill()
            stderr_out = proc.stderr.read(4096).decode(errors="replace")
            print(json.dumps({"success": False, "error": f"No response from server. stderr: {stderr_out[:500]}"}))
            sys.exit(1)

        body = b""
        while len(body) < content_length and time.time() < deadline:
            chunk = proc.stdout.read(content_length - len(body))
            if not chunk:
                break
            body += chunk

        resp = json.loads(body.decode())
        server_info = resp.get("result", {}).get("serverInfo", {})

        notif = json.dumps({"jsonrpc": "2.0", "method": "notifications/initialized"})
        notif_msg = f"Content-Length: {len(notif)}\r\n\r\n{notif}"
        try:
            proc.stdin.write(notif_msg.encode())
            proc.stdin.flush()
        except Exception:
            pass

        proc.kill()
        print(json.dumps({"success": True, "serverInfo": server_info}))

    except Exception as e:
        proc.kill()
        print(json.dumps({"success": False, "error": f"Protocol error: {e}"}))
        sys.exit(1)

main()
`

type mcpTestRequest struct {
	Command string            `json:"command" binding:"required"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env"`
}

type mcpTestResponse struct {
	Valid      bool                   `json:"valid"`
	ServerInfo map[string]interface{} `json:"serverInfo,omitempty"`
	Error      string                 `json:"error,omitempty"`
}

// TestMcpServer handles POST /api/projects/:projectName/mcp-config/test
// Spawns a temporary Pod using the runner image to test an MCP server connection.
func TestMcpServer(c *gin.Context) {
	var req mcpTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	projectName := c.Param("projectName")
	k8sClient, _ := GetK8sClientsForRequest(c)
	if k8sClient == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or missing token"})
		c.Abort()
		return
	}

	argsJSON, _ := json.Marshal(req.Args)
	envJSON, _ := json.Marshal(req.Env)

	runnerImage := os.Getenv("AMBIENT_CODE_RUNNER_IMAGE")
	if runnerImage == "" {
		runnerImage = "quay.io/ambient_code/vteam_claude_runner:latest"
	}

	podName := fmt.Sprintf("mcp-test-%s", rand.String(8))

	envVars := []corev1.EnvVar{
		{Name: "MCP_TEST_COMMAND", Value: req.Command},
		{Name: "MCP_TEST_ARGS", Value: string(argsJSON)},
		{Name: "MCP_TEST_ENV", Value: string(envJSON)},
	}
	for k, v := range req.Env {
		envVars = append(envVars, corev1.EnvVar{Name: k, Value: v})
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: projectName,
			Labels: map[string]string{
				"app":                      "mcp-test",
				"ambient-code.io/mcp-test": "true",
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			Containers: []corev1.Container{
				{
					Name:    "mcp-test",
					Image:   runnerImage,
					Command: []string{"python3", "-c", mcpTestScript},
					Env:     envVars,
				},
			},
		},
	}

	ctx := c.Request.Context()

	_, err := k8sClient.CoreV1().Pods(projectName).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		log.Printf("Failed to create MCP test pod in %s: %v", projectName, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("failed to create test pod: %v", err)})
		return
	}

	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = k8sClient.CoreV1().Pods(projectName).Delete(cleanupCtx, podName, metav1.DeleteOptions{})
	}()

	result, err := waitForMcpTestPod(ctx, k8sClient, projectName, podName)
	if err != nil {
		c.JSON(http.StatusOK, mcpTestResponse{Valid: false, Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

func waitForMcpTestPod(ctx context.Context, clientset kubernetes.Interface, namespace, podName string) (*mcpTestResponse, error) {
	timeout := time.After(60 * time.Second)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("request cancelled")
		case <-timeout:
			return nil, fmt.Errorf("test timed out after 60s")
		case <-ticker.C:
			pod, err := clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
			if err != nil {
				return nil, fmt.Errorf("failed to get pod status: %v", err)
			}
			switch pod.Status.Phase {
			case corev1.PodSucceeded, corev1.PodFailed:
				return readMcpTestLogs(ctx, clientset, namespace, podName)
			}
		}
	}
}

func readMcpTestLogs(ctx context.Context, clientset kubernetes.Interface, namespace, podName string) (*mcpTestResponse, error) {
	logReq := clientset.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{})
	logStream, err := logReq.Stream(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read pod logs: %v", err)
	}
	defer logStream.Close()

	buf := make([]byte, 8192)
	n, _ := logStream.Read(buf)
	logOutput := string(buf[:n])

	var podResult struct {
		Success    bool                   `json:"success"`
		ServerInfo map[string]interface{} `json:"serverInfo"`
		Error      string                 `json:"error"`
	}
	if err := json.Unmarshal([]byte(logOutput), &podResult); err != nil {
		return &mcpTestResponse{Valid: false, Error: fmt.Sprintf("failed to parse test output: %s", logOutput)}, nil
	}

	if podResult.Success {
		return &mcpTestResponse{Valid: true, ServerInfo: podResult.ServerInfo}, nil
	}
	return &mcpTestResponse{Valid: false, Error: podResult.Error}, nil
}
