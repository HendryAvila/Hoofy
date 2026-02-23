package memtools

import (
	"context"
	"fmt"

	"github.com/HendryAvila/sdd-hoffy/internal/memory"
	"github.com/mark3labs/mcp-go/mcp"
)

// SaveTool handles the mem_save MCP tool.
type SaveTool struct {
	store *memory.Store
}

// NewSaveTool creates a SaveTool with the given memory store.
func NewSaveTool(store *memory.Store) *SaveTool {
	return &SaveTool{store: store}
}

// Definition returns the MCP tool definition for mem_save.
func (t *SaveTool) Definition() mcp.Tool {
	return mcp.NewTool("mem_save",
		mcp.WithDescription(
			"Save an important observation to persistent memory. Call this PROACTIVELY after completing significant work — "+
				"don't wait to be asked. Save architectural decisions, bug fixes, new patterns, config changes, discoveries, gotchas.",
		),
		mcp.WithString("title",
			mcp.Required(),
			mcp.Description("Short, searchable title (e.g. 'JWT auth middleware', 'Fixed N+1 query')"),
		),
		mcp.WithString("content",
			mcp.Required(),
			mcp.Description("Structured content using **What**, **Why**, **Where**, **Learned** format"),
		),
		mcp.WithString("type",
			mcp.Description("Category: decision, architecture, bugfix, pattern, config, discovery, learning (default: manual)"),
		),
		mcp.WithString("session_id",
			mcp.Description("Session ID to associate with (default: manual-save)"),
		),
		mcp.WithString("project",
			mcp.Description("Project name"),
		),
		mcp.WithString("scope",
			mcp.Description("Scope for this observation: project (default) or personal"),
		),
		mcp.WithString("topic_key",
			mcp.Description("Optional topic identifier for upserts (e.g. architecture/auth-model). Reuses and updates the latest observation in same project+scope."),
		),
	)
}

// Handle processes the mem_save tool call.
func (t *SaveTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	title := req.GetString("title", "")
	content := req.GetString("content", "")

	if title == "" {
		return mcp.NewToolResultError("'title' is required"), nil
	}
	if content == "" {
		return mcp.NewToolResultError("'content' is required"), nil
	}

	sessionID := req.GetString("session_id", "manual-save")
	typ := req.GetString("type", "manual")
	project := req.GetString("project", "")
	scope := req.GetString("scope", "project")
	topicKey := req.GetString("topic_key", "")

	id, err := t.store.AddObservation(memory.AddObservationParams{
		SessionID: sessionID,
		Type:      typ,
		Title:     title,
		Content:   content,
		Project:   project,
		Scope:     scope,
		TopicKey:  topicKey,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to save observation: %v", err)), nil
	}

	response := fmt.Sprintf("Memory saved: %q (%s)", title, typ)
	if topicKey == "" {
		suggested := memory.SuggestTopicKey(typ, title, content)
		response += fmt.Sprintf("\nSuggested topic_key: %s", suggested)
	}
	response += fmt.Sprintf("\nID: %d", id)

	return mcp.NewToolResultText(response), nil
}

// ─── SavePromptTool ─────────────────────────────────────────────────────────

// SavePromptTool handles the mem_save_prompt MCP tool.
type SavePromptTool struct {
	store *memory.Store
}

// NewSavePromptTool creates a SavePromptTool.
func NewSavePromptTool(store *memory.Store) *SavePromptTool {
	return &SavePromptTool{store: store}
}

// Definition returns the MCP tool definition for mem_save_prompt.
func (t *SavePromptTool) Definition() mcp.Tool {
	return mcp.NewTool("mem_save_prompt",
		mcp.WithDescription(
			"Save a user prompt to persistent memory. Use this to record what the user asked — "+
				"their intent, questions, and requests — so future sessions have context about the user's goals.",
		),
		mcp.WithString("content",
			mcp.Required(),
			mcp.Description("The user's prompt text"),
		),
		mcp.WithString("session_id",
			mcp.Description("Session ID to associate with (default: manual-save)"),
		),
		mcp.WithString("project",
			mcp.Description("Project name"),
		),
	)
}

// Handle processes the mem_save_prompt tool call.
func (t *SavePromptTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	content := req.GetString("content", "")
	if content == "" {
		return mcp.NewToolResultError("'content' is required"), nil
	}

	sessionID := req.GetString("session_id", "manual-save")
	project := req.GetString("project", "")

	id, err := t.store.AddPrompt(memory.AddPromptParams{
		SessionID: sessionID,
		Content:   content,
		Project:   project,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to save prompt: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Prompt saved (ID: %d)", id)), nil
}
