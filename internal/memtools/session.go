package memtools

import (
	"context"
	"fmt"

	"github.com/HendryAvila/sdd-hoffy/internal/memory"
	"github.com/mark3labs/mcp-go/mcp"
)

// SessionStartTool handles the mem_session_start MCP tool.
type SessionStartTool struct {
	store *memory.Store
}

// NewSessionStartTool creates a SessionStartTool.
func NewSessionStartTool(store *memory.Store) *SessionStartTool {
	return &SessionStartTool{store: store}
}

// Definition returns the MCP tool definition for mem_session_start.
func (t *SessionStartTool) Definition() mcp.Tool {
	return mcp.NewTool("mem_session_start",
		mcp.WithDescription(
			"Register the start of a new coding session. Call this at the beginning "+
				"of a session to track activity.",
		),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Unique session identifier"),
		),
		mcp.WithString("project",
			mcp.Required(),
			mcp.Description("Project name"),
		),
		mcp.WithString("directory",
			mcp.Description("Working directory"),
		),
	)
}

// Handle processes the mem_session_start tool call.
func (t *SessionStartTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetString("id", "")
	project := req.GetString("project", "")

	if id == "" {
		return mcp.NewToolResultError("'id' is required"), nil
	}
	if project == "" {
		return mcp.NewToolResultError("'project' is required"), nil
	}

	directory := req.GetString("directory", "")

	if err := t.store.CreateSession(id, project, directory); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to start session: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Session %q started for project %q", id, project)), nil
}

// ─── SessionEndTool ─────────────────────────────────────────────────────────

// SessionEndTool handles the mem_session_end MCP tool.
type SessionEndTool struct {
	store *memory.Store
}

// NewSessionEndTool creates a SessionEndTool.
func NewSessionEndTool(store *memory.Store) *SessionEndTool {
	return &SessionEndTool{store: store}
}

// Definition returns the MCP tool definition for mem_session_end.
func (t *SessionEndTool) Definition() mcp.Tool {
	return mcp.NewTool("mem_session_end",
		mcp.WithDescription(
			"Mark a coding session as completed with an optional summary.",
		),
		mcp.WithString("id",
			mcp.Required(),
			mcp.Description("Session identifier to close"),
		),
		mcp.WithString("summary",
			mcp.Description("Summary of what was accomplished"),
		),
	)
}

// Handle processes the mem_session_end tool call.
func (t *SessionEndTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := req.GetString("id", "")
	if id == "" {
		return mcp.NewToolResultError("'id' is required"), nil
	}

	summary := req.GetString("summary", "")

	if err := t.store.EndSession(id, summary); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to end session: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Session %q completed", id)), nil
}

// ─── SessionSummaryTool ─────────────────────────────────────────────────────

// SessionSummaryTool handles the mem_session_summary MCP tool.
type SessionSummaryTool struct {
	store *memory.Store
}

// NewSessionSummaryTool creates a SessionSummaryTool.
func NewSessionSummaryTool(store *memory.Store) *SessionSummaryTool {
	return &SessionSummaryTool{store: store}
}

// Definition returns the MCP tool definition for mem_session_summary.
func (t *SessionSummaryTool) Definition() mcp.Tool {
	return mcp.NewTool("mem_session_summary",
		mcp.WithDescription(
			"Save a comprehensive end-of-session summary. Call this when a session is ending "+
				"or when significant work is complete. This creates a structured summary that future "+
				"sessions will use to understand what happened.",
		),
		mcp.WithString("content",
			mcp.Required(),
			mcp.Description("Full session summary using Goal/Instructions/Discoveries/Accomplished/Files format"),
		),
		mcp.WithString("project",
			mcp.Required(),
			mcp.Description("Project name"),
		),
		mcp.WithString("session_id",
			mcp.Description("Session ID (default: manual-save)"),
		),
	)
}

// Handle processes the mem_session_summary tool call.
func (t *SessionSummaryTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	content := req.GetString("content", "")
	project := req.GetString("project", "")

	if content == "" {
		return mcp.NewToolResultError("'content' is required"), nil
	}
	if project == "" {
		return mcp.NewToolResultError("'project' is required"), nil
	}

	sessionID := req.GetString("session_id", "manual-save")

	id, err := t.store.AddObservation(memory.AddObservationParams{
		SessionID: sessionID,
		Type:      "session_summary",
		Title:     fmt.Sprintf("Session summary: %s", project),
		Content:   content,
		Project:   project,
		Scope:     "project",
		TopicKey:  "",
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to save session summary: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Session summary saved for %q (ID: %d)", project, id)), nil
}
