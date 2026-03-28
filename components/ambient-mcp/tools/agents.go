package tools

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/ambient-code/platform/components/ambient-mcp/client"
)

type agentList struct {
	Kind  string  `json:"kind"`
	Page  int     `json:"page"`
	Size  int     `json:"size"`
	Total int     `json:"total"`
	Items []agent `json:"items"`
}

type agent struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	ProjectID   string `json:"project_id,omitempty"`
	Prompt      string `json:"prompt,omitempty"`
	Labels      string `json:"labels,omitempty"`
	Annotations string `json:"annotations,omitempty"`
	Version     int    `json:"version,omitempty"`
}

func ListAgents(c *client.Client) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		projectID := mcp.ParseString(req, "project_id", "")
		if projectID == "" {
			return errResult("INVALID_REQUEST", "project_id is required"), nil
		}

		params := url.Values{}
		if v := mcp.ParseString(req, "search", ""); v != "" {
			params.Set("search", v)
		}
		page := mcp.ParseInt(req, "page", 0)
		if page > 0 {
			params.Set("page", fmt.Sprintf("%d", page))
		}
		size := mcp.ParseInt(req, "size", 0)
		if size > 0 {
			params.Set("size", fmt.Sprintf("%d", size))
		}

		var result agentList
		path := "/projects/" + url.PathEscape(projectID) + "/agents"
		if err := c.GetWithQuery(ctx, path, params, &result); err != nil {
			return errResult("LIST_FAILED", err.Error()), nil
		}
		return jsonResult(result)
	}
}

func GetAgent(c *client.Client) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		projectID := mcp.ParseString(req, "project_id", "")
		if projectID == "" {
			return errResult("INVALID_REQUEST", "project_id is required"), nil
		}
		agentID := mcp.ParseString(req, "agent_id", "")
		if agentID == "" {
			return errResult("INVALID_REQUEST", "agent_id is required"), nil
		}

		path := "/projects/" + url.PathEscape(projectID) + "/agents/" + url.PathEscape(agentID)
		var result agent
		if err := c.Get(ctx, path, &result); err != nil {
			return errResult("AGENT_NOT_FOUND", err.Error()), nil
		}
		return jsonResult(result)
	}
}

func CreateAgent(c *client.Client) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		projectID := mcp.ParseString(req, "project_id", "")
		if projectID == "" {
			return errResult("INVALID_REQUEST", "project_id is required"), nil
		}
		name := mcp.ParseString(req, "name", "")
		if name == "" {
			return errResult("INVALID_REQUEST", "name is required"), nil
		}
		prompt := mcp.ParseString(req, "prompt", "")
		if prompt == "" {
			return errResult("INVALID_REQUEST", "prompt is required"), nil
		}

		body := map[string]interface{}{
			"name":   name,
			"prompt": prompt,
		}
		path := "/projects/" + url.PathEscape(projectID) + "/agents"
		var result agent
		if err := c.Post(ctx, path, body, &result, http.StatusCreated); err != nil {
			return errResult("CREATE_FAILED", err.Error()), nil
		}
		return jsonResult(result)
	}
}

func UpdateAgent(c *client.Client) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		projectID := mcp.ParseString(req, "project_id", "")
		if projectID == "" {
			return errResult("INVALID_REQUEST", "project_id is required"), nil
		}
		agentID := mcp.ParseString(req, "agent_id", "")
		if agentID == "" {
			return errResult("INVALID_REQUEST", "agent_id is required"), nil
		}

		patch := map[string]interface{}{}
		if v := mcp.ParseString(req, "prompt", ""); v != "" {
			patch["prompt"] = v
		}
		if v := mcp.ParseStringMap(req, "labels", nil); v != nil {
			patch["labels"] = v
		}
		if v := mcp.ParseStringMap(req, "annotations", nil); v != nil {
			patch["annotations"] = v
		}
		if len(patch) == 0 {
			return errResult("INVALID_REQUEST", "at least one of prompt, labels, or annotations must be provided"), nil
		}

		path := "/projects/" + url.PathEscape(projectID) + "/agents/" + url.PathEscape(agentID)
		var result agent
		if err := c.Patch(ctx, path, patch, &result); err != nil {
			return errResult("UPDATE_FAILED", err.Error()), nil
		}
		return jsonResult(result)
	}
}

func PatchAgentAnnotations(c *client.Client) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		projectID := mcp.ParseString(req, "project_id", "")
		if projectID == "" {
			return errResult("INVALID_REQUEST", "project_id is required"), nil
		}
		agentID := mcp.ParseString(req, "agent_id", "")
		if agentID == "" {
			return errResult("INVALID_REQUEST", "agent_id is required"), nil
		}

		annRaw := mcp.ParseStringMap(req, "annotations", nil)
		if annRaw == nil {
			return errResult("INVALID_REQUEST", "annotations is required"), nil
		}

		patch := make(map[string]string, len(annRaw))
		for k, v := range annRaw {
			s, ok := v.(string)
			if !ok {
				return errResult("INVALID_REQUEST", fmt.Sprintf("annotation %q: value must be a string", k)), nil
			}
			patch[k] = s
		}

		path := "/projects/" + url.PathEscape(projectID) + "/agents/" + url.PathEscape(agentID)
		var existing agent
		if err := c.Get(ctx, path, &existing); err != nil {
			return errResult("AGENT_NOT_FOUND", err.Error()), nil
		}

		merged := mergeStringMaps(existing.Annotations, patch)

		var result agent
		if err := c.Patch(ctx, path, map[string]interface{}{"annotations": merged}, &result); err != nil {
			return errResult("PATCH_FAILED", err.Error()), nil
		}
		return jsonResult(result)
	}
}
