package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/ambient-code/platform/components/ambient-mcp/client"
	"github.com/ambient-code/platform/components/ambient-mcp/mention"
)

type sessionList struct {
	Kind  string    `json:"kind"`
	Page  int       `json:"page"`
	Size  int       `json:"size"`
	Total int       `json:"total"`
	Items []session `json:"items"`
}

type session struct {
	ID              string `json:"id,omitempty"`
	Name            string `json:"name,omitempty"`
	ProjectID       string `json:"project_id,omitempty"`
	Phase           string `json:"phase,omitempty"`
	Prompt          string `json:"prompt,omitempty"`
	AgentID         string `json:"agent_id,omitempty"`
	ParentSessionID string `json:"parent_session_id,omitempty"`
	LlmModel        string `json:"llm_model,omitempty"`
	Labels          string `json:"labels,omitempty"`
	Annotations     string `json:"annotations,omitempty"`
	CreatedAt       string `json:"created_at,omitempty"`
}

type sessionMessage struct {
	ID        string `json:"id,omitempty"`
	SessionID string `json:"session_id,omitempty"`
	Seq       int    `json:"seq,omitempty"`
	EventType string `json:"event_type,omitempty"`
	Payload   string `json:"payload,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

func ListSessions(c *client.Client) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		params := url.Values{}
		if v := mcp.ParseString(req, "project_id", ""); v != "" {
			params.Set("search", "project_id = '"+v+"'")
		}
		if v := mcp.ParseString(req, "phase", ""); v != "" {
			existing := params.Get("search")
			filter := "phase = '" + v + "'"
			if existing != "" {
				params.Set("search", existing+" and "+filter)
			} else {
				params.Set("search", filter)
			}
		}
		page := mcp.ParseInt(req, "page", 0)
		if page > 0 {
			params.Set("page", fmt.Sprintf("%d", page))
		}
		size := mcp.ParseInt(req, "size", 0)
		if size > 0 {
			params.Set("size", fmt.Sprintf("%d", size))
		}

		var result sessionList
		if err := c.GetWithQuery(ctx, "/sessions", params, &result); err != nil {
			return errResult("SESSION_LIST_FAILED", err.Error()), nil
		}
		return jsonResult(result)
	}
}

func GetSession(c *client.Client) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		id := mcp.ParseString(req, "session_id", "")
		if id == "" {
			return errResult("INVALID_REQUEST", "session_id is required"), nil
		}
		var result session
		if err := c.Get(ctx, "/sessions/"+url.PathEscape(id), &result); err != nil {
			return errResult("SESSION_NOT_FOUND", err.Error()), nil
		}
		return jsonResult(result)
	}
}

func CreateSession(c *client.Client) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		projectID := mcp.ParseString(req, "project_id", "")
		if projectID == "" {
			return errResult("INVALID_REQUEST", "project_id is required"), nil
		}
		prompt := mcp.ParseString(req, "prompt", "")
		if prompt == "" {
			return errResult("INVALID_REQUEST", "prompt is required"), nil
		}

		body := map[string]interface{}{
			"project_id": projectID,
			"prompt":     prompt,
		}
		if v := mcp.ParseString(req, "agent_id", ""); v != "" {
			body["agent_id"] = v
		}
		if v := mcp.ParseString(req, "model", ""); v != "" {
			body["llm_model"] = v
		}
		if v := mcp.ParseString(req, "parent_session_id", ""); v != "" {
			body["parent_session_id"] = v
		}
		if v := mcp.ParseString(req, "name", ""); v != "" {
			body["name"] = v
		}

		var created session
		if err := c.Post(ctx, "/sessions", body, &created, http.StatusCreated); err != nil {
			return errResult("CREATE_FAILED", err.Error()), nil
		}

		var started session
		if err := c.Post(ctx, "/sessions/"+url.PathEscape(created.ID)+"/start", nil, &started, http.StatusOK); err != nil {
			return errResult("START_FAILED", err.Error()), nil
		}
		return jsonResult(started)
	}
}

func PushMessage(c *client.Client) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	resolver, err := mention.NewResolver(c.BaseURL(), c.Token)
	if err != nil {
		return func(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return errResult("CONFIG_ERROR", err.Error()), nil
		}
	}

	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		sessionID := mcp.ParseString(req, "session_id", "")
		if sessionID == "" {
			return errResult("INVALID_REQUEST", "session_id is required"), nil
		}
		text := mcp.ParseString(req, "text", "")
		if text == "" {
			return errResult("INVALID_REQUEST", "text is required"), nil
		}

		body := map[string]interface{}{"payload": text}
		var pushed sessionMessage
		if err := c.Post(ctx, "/sessions/"+url.PathEscape(sessionID)+"/messages", body, &pushed, http.StatusCreated); err != nil {
			return errResult("PUSH_FAILED", err.Error()), nil
		}

		var callerSession session
		if err := c.Get(ctx, "/sessions/"+url.PathEscape(sessionID), &callerSession); err != nil {
			return errResult("SESSION_NOT_FOUND", err.Error()), nil
		}

		matches := mention.Extract(text)
		var delegated interface{}
		for _, m := range matches {
			agentID, err := resolver.Resolve(ctx, callerSession.ProjectID, m.Identifier)
			if err != nil {
				return errResult("MENTION_NOT_RESOLVED", err.Error()), nil
			}
			stripped := mention.StripToken(text, m.Token)
			createBody := map[string]interface{}{
				"project_id":        callerSession.ProjectID,
				"prompt":            stripped,
				"agent_id":          agentID,
				"parent_session_id": sessionID,
			}
			var child session
			if err := c.Post(ctx, "/sessions", createBody, &child, http.StatusCreated); err != nil {
				return errResult("DELEGATION_FAILED", err.Error()), nil
			}
			var started session
			if err := c.Post(ctx, "/sessions/"+url.PathEscape(child.ID)+"/start", nil, &started, http.StatusOK); err != nil {
				return errResult("DELEGATION_START_FAILED", err.Error()), nil
			}
			delegated = started
			break
		}

		response := map[string]interface{}{
			"message":           pushed,
			"delegated_session": delegated,
		}
		return jsonResult(response)
	}
}

func PatchSessionLabels(c *client.Client) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		sessionID := mcp.ParseString(req, "session_id", "")
		if sessionID == "" {
			return errResult("INVALID_REQUEST", "session_id is required"), nil
		}

		labelsRaw := mcp.ParseStringMap(req, "labels", nil)
		if labelsRaw == nil {
			return errResult("INVALID_REQUEST", "labels is required"), nil
		}

		labels := make(map[string]string, len(labelsRaw))
		for k, v := range labelsRaw {
			s, ok := v.(string)
			if !ok {
				return errResult("INVALID_LABEL_VALUE", fmt.Sprintf("label %q: value must be a string", k)), nil
			}
			labels[k] = s
		}

		var existing session
		if err := c.Get(ctx, "/sessions/"+url.PathEscape(sessionID), &existing); err != nil {
			return errResult("SESSION_NOT_FOUND", err.Error()), nil
		}

		merged := mergeStringMaps(existing.Labels, labels)

		var result session
		if err := c.Patch(ctx, "/sessions/"+url.PathEscape(sessionID), map[string]interface{}{"labels": merged}, &result); err != nil {
			return errResult("PATCH_FAILED", err.Error()), nil
		}
		return jsonResult(result)
	}
}

func PatchSessionAnnotations(c *client.Client) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		sessionID := mcp.ParseString(req, "session_id", "")
		if sessionID == "" {
			return errResult("INVALID_REQUEST", "session_id is required"), nil
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

		var existing session
		if err := c.Get(ctx, "/sessions/"+url.PathEscape(sessionID), &existing); err != nil {
			return errResult("SESSION_NOT_FOUND", err.Error()), nil
		}

		merged := mergeStringMaps(existing.Annotations, patch)

		var result session
		if err := c.Patch(ctx, "/sessions/"+url.PathEscape(sessionID), map[string]interface{}{"annotations": merged}, &result); err != nil {
			return errResult("PATCH_FAILED", err.Error()), nil
		}
		return jsonResult(result)
	}
}

func mergeStringMaps(existingJSON string, patch map[string]string) string {
	merged := make(map[string]string)
	if existingJSON != "" {
		_ = json.Unmarshal([]byte(existingJSON), &merged)
	}
	for k, v := range patch {
		if v == "" {
			delete(merged, k)
		} else {
			merged[k] = v
		}
	}
	b, _ := json.Marshal(merged)
	return string(b)
}
