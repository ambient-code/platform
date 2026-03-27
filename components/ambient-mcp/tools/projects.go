package tools

import (
	"context"
	"fmt"
	"net/url"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/ambient-code/platform/components/ambient-mcp/client"
)

type projectList struct {
	Kind  string    `json:"kind"`
	Page  int       `json:"page"`
	Size  int       `json:"size"`
	Total int       `json:"total"`
	Items []project `json:"items"`
}

type project struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Labels      string `json:"labels,omitempty"`
	Annotations string `json:"annotations,omitempty"`
	Prompt      string `json:"prompt,omitempty"`
	Status      string `json:"status,omitempty"`
}

func ListProjects(c *client.Client) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := url.Values{}
		page := mcp.ParseInt(req, "page", 0)
		if page > 0 {
			params.Set("page", fmt.Sprintf("%d", page))
		}
		size := mcp.ParseInt(req, "size", 0)
		if size > 0 {
			params.Set("size", fmt.Sprintf("%d", size))
		}

		var result projectList
		if err := c.GetWithQuery(ctx, "/projects", params, &result); err != nil {
			return errResult("LIST_FAILED", err.Error()), nil
		}
		return jsonResult(result)
	}
}

func GetProject(c *client.Client) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		projectID := mcp.ParseString(req, "project_id", "")
		if projectID == "" {
			return errResult("INVALID_REQUEST", "project_id is required"), nil
		}

		var result project
		if err := c.Get(ctx, "/projects/"+url.PathEscape(projectID), &result); err != nil {
			return errResult("PROJECT_NOT_FOUND", err.Error()), nil
		}
		return jsonResult(result)
	}
}

func PatchProjectAnnotations(c *client.Client) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		projectID := mcp.ParseString(req, "project_id", "")
		if projectID == "" {
			return errResult("INVALID_REQUEST", "project_id is required"), nil
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

		path := "/projects/" + url.PathEscape(projectID)
		var existing project
		if err := c.Get(ctx, path, &existing); err != nil {
			return errResult("PROJECT_NOT_FOUND", err.Error()), nil
		}

		merged := mergeStringMaps(existing.Annotations, patch)

		var result project
		if err := c.Patch(ctx, path, map[string]interface{}{"annotations": merged}, &result); err != nil {
			return errResult("PATCH_FAILED", err.Error()), nil
		}
		return jsonResult(result)
	}
}
