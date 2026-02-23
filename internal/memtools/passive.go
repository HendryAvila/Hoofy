package memtools

import (
	"context"
	"fmt"

	"github.com/HendryAvila/Hoofy/internal/memory"
	"github.com/mark3labs/mcp-go/mcp"
)

// PassiveCaptureTool handles the mem_capture_passive MCP tool.
type PassiveCaptureTool struct {
	store *memory.Store
}

// NewPassiveCaptureTool creates a PassiveCaptureTool with the given memory store.
func NewPassiveCaptureTool(store *memory.Store) *PassiveCaptureTool {
	return &PassiveCaptureTool{store: store}
}

// Definition returns the MCP tool definition for mem_capture_passive.
func (t *PassiveCaptureTool) Definition() mcp.Tool {
	return mcp.NewTool("mem_capture_passive",
		mcp.WithDescription(
			"Passively capture learnings from conversation content. Extracts key insights, decisions, "+
				"and patterns automatically without explicit user action. Use this to silently capture "+
				"important context from ongoing work.",
		),
		mcp.WithString("content",
			mcp.Required(),
			mcp.Description("The conversation or work content to extract learnings from"),
		),
		mcp.WithString("session_id",
			mcp.Description("Session ID to associate captured learnings with (default: manual-save)"),
		),
		mcp.WithString("project",
			mcp.Description("Project name"),
		),
		mcp.WithString("source",
			mcp.Description("Source identifier for the captured content (e.g. tool name)"),
		),
	)
}

// Handle processes the mem_capture_passive tool call.
func (t *PassiveCaptureTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	content := req.GetString("content", "")
	if content == "" {
		return mcp.NewToolResultError("'content' is required"), nil
	}

	sessionID := req.GetString("session_id", "manual-save")
	project := req.GetString("project", "")
	source := req.GetString("source", "")

	result, err := t.store.PassiveCapture(memory.PassiveCaptureParams{
		SessionID: sessionID,
		Content:   content,
		Project:   project,
		Source:    source,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("passive capture failed: %v", err)), nil
	}

	response := fmt.Sprintf("Passive capture complete: %d extracted, %d saved, %d duplicates",
		result.Extracted, result.Saved, result.Duplicates)

	return mcp.NewToolResultText(response), nil
}
