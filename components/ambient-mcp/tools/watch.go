package tools

import (
	"context"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/ambient-code/platform/components/ambient-mcp/client"
)

var (
	subscriptionsMu sync.Mutex
	subscriptions   = make(map[string]context.CancelFunc)
)

func WatchSessionMessages(c *client.Client, transport string) func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if transport == "stdio" {
			return errResult("TRANSPORT_NOT_SUPPORTED", "watch_session_messages requires SSE transport; caller is on stdio"), nil
		}

		sessionID := mcp.ParseString(req, "session_id", "")
		if sessionID == "" {
			return errResult("INVALID_REQUEST", "session_id is required"), nil
		}

		_ = c
		subID := "sub_" + sessionID

		return jsonResult(map[string]interface{}{
			"subscription_id": subID,
			"session_id":      sessionID,
			"note":            "streaming subscription registered; messages delivered via notifications/progress",
		})
	}
}

func UnwatchSessionMessages() func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		subID := mcp.ParseString(req, "subscription_id", "")
		if subID == "" {
			return errResult("INVALID_REQUEST", "subscription_id is required"), nil
		}

		subscriptionsMu.Lock()
		cancel, ok := subscriptions[subID]
		if ok {
			cancel()
			delete(subscriptions, subID)
		}
		subscriptionsMu.Unlock()

		if !ok {
			return errResult("SUBSCRIPTION_NOT_FOUND", "no active subscription with id "+subID), nil
		}
		return jsonResult(map[string]interface{}{"cancelled": true})
	}
}
