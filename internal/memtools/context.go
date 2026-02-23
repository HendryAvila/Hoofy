package memtools

import (
	"context"

	"github.com/HendryAvila/Hoofy/internal/memory"
	"github.com/mark3labs/mcp-go/mcp"
)

// ContextTool handles the mem_context MCP tool.
type ContextTool struct {
	store *memory.Store
}

// NewContextTool creates a ContextTool.
func NewContextTool(store *memory.Store) *ContextTool {
	return &ContextTool{store: store}
}

// Definition returns the MCP tool definition for mem_context.
func (t *ContextTool) Definition() mcp.Tool {
	return mcp.NewTool("mem_context",
		mcp.WithDescription(
			"Get recent memory context from previous sessions. Shows recent sessions and "+
				"observations to understand what was done before.",
		),
		mcp.WithString("project",
			mcp.Description("Filter by project (omit for all projects)"),
		),
		mcp.WithString("scope",
			mcp.Description("Filter observations by scope: project (default) or personal"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Number of observations to retrieve (default: 20)"),
		),
	)
}

// Handle processes the mem_context tool call.
func (t *ContextTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	project := req.GetString("project", "")
	scope := req.GetString("scope", "")

	formatted, err := t.store.FormatContext(project, scope)
	if err != nil {
		return mcp.NewToolResultText("No memory context available."), nil
	}

	if formatted == "" {
		return mcp.NewToolResultText("No memory context available yet. Start saving observations with mem_save."), nil
	}

	return mcp.NewToolResultText(formatted), nil
}
