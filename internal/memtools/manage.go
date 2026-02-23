package memtools

import (
	"context"
	"fmt"

	"github.com/HendryAvila/sdd-hoffy/internal/memory"
	"github.com/mark3labs/mcp-go/mcp"
)

// ─── DeleteTool ─────────────────────────────────────────────────────────────

// DeleteTool handles the mem_delete MCP tool.
type DeleteTool struct {
	store *memory.Store
}

// NewDeleteTool creates a DeleteTool with the given memory store.
func NewDeleteTool(store *memory.Store) *DeleteTool {
	return &DeleteTool{store: store}
}

// Definition returns the MCP tool definition for mem_delete.
func (t *DeleteTool) Definition() mcp.Tool {
	return mcp.NewTool("mem_delete",
		mcp.WithDescription(
			"Delete an observation by ID. Soft-delete by default; set hard_delete=true for permanent deletion.",
		),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("Observation ID to delete"),
		),
		mcp.WithBoolean("hard_delete",
			mcp.Description("If true, permanently deletes the observation"),
		),
	)
}

// Handle processes the mem_delete tool call.
func (t *DeleteTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := intArg(req, "id", 0)
	if id == 0 {
		return mcp.NewToolResultError("'id' is required"), nil
	}

	hardDelete := boolArg(req, "hard_delete", false)

	err := t.store.DeleteObservation(int64(id), hardDelete)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to delete observation: %v", err)), nil
	}

	action := "soft-deleted"
	if hardDelete {
		action = "permanently deleted"
	}
	return mcp.NewToolResultText(fmt.Sprintf("Observation %d %s", id, action)), nil
}

// ─── UpdateTool ─────────────────────────────────────────────────────────────

// UpdateTool handles the mem_update MCP tool.
type UpdateTool struct {
	store *memory.Store
}

// NewUpdateTool creates an UpdateTool with the given memory store.
func NewUpdateTool(store *memory.Store) *UpdateTool {
	return &UpdateTool{store: store}
}

// Definition returns the MCP tool definition for mem_update.
func (t *UpdateTool) Definition() mcp.Tool {
	return mcp.NewTool("mem_update",
		mcp.WithDescription(
			"Update an existing observation by ID. Only provided fields are changed.",
		),
		mcp.WithNumber("id",
			mcp.Required(),
			mcp.Description("Observation ID to update"),
		),
		mcp.WithString("title",
			mcp.Description("New title"),
		),
		mcp.WithString("content",
			mcp.Description("New content"),
		),
		mcp.WithString("type",
			mcp.Description("New type/category"),
		),
		mcp.WithString("project",
			mcp.Description("New project value"),
		),
		mcp.WithString("scope",
			mcp.Description("New scope: project or personal"),
		),
		mcp.WithString("topic_key",
			mcp.Description("New topic key (normalized internally)"),
		),
	)
}

// Handle processes the mem_update tool call.
func (t *UpdateTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	id := intArg(req, "id", 0)
	if id == 0 {
		return mcp.NewToolResultError("'id' is required"), nil
	}

	params := memory.UpdateObservationParams{}
	hasUpdates := false

	if v := req.GetString("title", ""); v != "" {
		params.Title = &v
		hasUpdates = true
	}
	if v := req.GetString("content", ""); v != "" {
		params.Content = &v
		hasUpdates = true
	}
	if v := req.GetString("type", ""); v != "" {
		params.Type = &v
		hasUpdates = true
	}
	if v := req.GetString("project", ""); v != "" {
		params.Project = &v
		hasUpdates = true
	}
	if v := req.GetString("scope", ""); v != "" {
		params.Scope = &v
		hasUpdates = true
	}
	if v := req.GetString("topic_key", ""); v != "" {
		params.TopicKey = &v
		hasUpdates = true
	}

	if !hasUpdates {
		return mcp.NewToolResultError("at least one field to update is required"), nil
	}

	obs, err := t.store.UpdateObservation(int64(id), params)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to update observation: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Observation %d updated: %q (rev %d)", obs.ID, obs.Title, obs.RevisionCount)), nil
}

// ─── SuggestTopicKeyTool ────────────────────────────────────────────────────

// SuggestTopicKeyTool handles the mem_suggest_topic_key MCP tool.
type SuggestTopicKeyTool struct{}

// NewSuggestTopicKeyTool creates a SuggestTopicKeyTool.
// This tool is stateless — it doesn't need the store.
func NewSuggestTopicKeyTool() *SuggestTopicKeyTool {
	return &SuggestTopicKeyTool{}
}

// Definition returns the MCP tool definition for mem_suggest_topic_key.
func (t *SuggestTopicKeyTool) Definition() mcp.Tool {
	return mcp.NewTool("mem_suggest_topic_key",
		mcp.WithDescription(
			"Suggest a stable topic_key for memory upserts. Use this before mem_save when you want evolving topics "+
				"(like architecture decisions) to update a single observation over time.",
		),
		mcp.WithString("title",
			mcp.Description("Observation title (preferred input for stable keys)"),
		),
		mcp.WithString("content",
			mcp.Description("Observation content used as fallback if title is empty"),
		),
		mcp.WithString("type",
			mcp.Description("Observation type/category, e.g. architecture, decision, bugfix"),
		),
	)
}

// Handle processes the mem_suggest_topic_key tool call.
func (t *SuggestTopicKeyTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	title := req.GetString("title", "")
	content := req.GetString("content", "")
	typ := req.GetString("type", "")

	if title == "" && content == "" {
		return mcp.NewToolResultError("at least 'title' or 'content' is required"), nil
	}

	key := memory.SuggestTopicKey(typ, title, content)
	return mcp.NewToolResultText(fmt.Sprintf("Suggested topic_key: %s", key)), nil
}
