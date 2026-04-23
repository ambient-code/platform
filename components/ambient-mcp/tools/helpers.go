package tools

import (
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"
)

func jsonResult(v interface{}) (*mcp.CallToolResult, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return mcp.NewToolResultError("marshal error: " + err.Error()), nil
	}
	return mcp.NewToolResultText(string(b)), nil
}

func errResult(code, reason string) *mcp.CallToolResult {
	b, _ := json.Marshal(map[string]string{
		"code":   code,
		"reason": reason,
	})
	return mcp.NewToolResultError(string(b))
}
