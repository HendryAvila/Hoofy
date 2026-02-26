package memtools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/HendryAvila/Hoofy/internal/memory"
	"github.com/mark3labs/mcp-go/mcp"
)

// ProgressTool handles the mem_progress MCP tool.
// It provides dual read/write behavior for intra-session progress tracking:
//   - Without content: reads the current progress for a project
//   - With content: upserts a structured JSON progress document
//
// Only ONE active progress exists per project (enforced via topic_key upsert).
type ProgressTool struct {
	store *memory.Store
}

// NewProgressTool creates a ProgressTool with the given memory store.
func NewProgressTool(store *memory.Store) *ProgressTool {
	return &ProgressTool{store: store}
}

// progressTopicKey returns the canonical topic key for a project's progress doc.
func progressTopicKey(project string) string {
	return "progress/" + project
}

// Definition returns the MCP tool definition for mem_progress.
func (t *ProgressTool) Definition() mcp.Tool {
	return mcp.NewTool("mem_progress",
		mcp.WithDescription(
			"Persist or retrieve a structured progress document for long-running sessions. "+
				"Dual behavior: call WITHOUT content to read current progress, WITH content to upsert it. "+
				"One active progress per project (auto-upserted via topic_key). "+
				"Content MUST be valid JSON with fields like: goal, completed, next_steps, blockers. "+
				"Use this to survive context window compaction — the progress doc persists across sessions.",
		),
		mcp.WithString("project",
			mcp.Required(),
			mcp.Description("Project name — determines which progress doc to read/write"),
		),
		mcp.WithString("content",
			mcp.Description(
				"JSON progress document to save. If omitted, reads current progress. "+
					"Recommended structure: {\"goal\": \"...\", \"completed\": [...], \"next_steps\": [...], \"blockers\": [...]}",
			),
		),
		mcp.WithString("session_id",
			mcp.Description("Session ID to associate with (default: manual-save)"),
		),
	)
}

// Handle processes the mem_progress tool call.
func (t *ProgressTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	project := req.GetString("project", "")
	if project == "" {
		return mcp.NewToolResultError("'project' is required"), nil
	}

	content := req.GetString("content", "")

	if content == "" {
		return t.handleRead(project)
	}
	return t.handleWrite(project, content, req.GetString("session_id", "manual-save"))
}

// handleRead retrieves the current progress document for a project.
func (t *ProgressTool) handleRead(project string) (*mcp.CallToolResult, error) {
	topicKey := progressTopicKey(project)
	obs, err := t.store.FindByTopicKey(topicKey, project, "project")
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to read progress: %v", err)), nil
	}
	if obs == nil {
		return mcp.NewToolResultText(
			fmt.Sprintf("No progress document found for project %q. "+
				"Call mem_progress with content to create one.", project),
		), nil
	}

	response := fmt.Sprintf("# Progress: %s\n\n", project)
	response += obs.Content
	response += fmt.Sprintf("\n\n---\n_Last updated: %s | ID: %d | Revisions: %d_", obs.UpdatedAt, obs.ID, obs.RevisionCount)

	return mcp.NewToolResultText(response), nil
}

// handleWrite validates and saves a JSON progress document.
func (t *ProgressTool) handleWrite(project, content, sessionID string) (*mcp.CallToolResult, error) {
	// Validate JSON
	if !json.Valid([]byte(content)) {
		return mcp.NewToolResultError(
			"'content' must be valid JSON. " +
				"Recommended: {\"goal\": \"...\", \"completed\": [...], \"next_steps\": [...], \"blockers\": [...]}",
		), nil
	}

	topicKey := progressTopicKey(project)
	id, err := t.store.AddObservation(memory.AddObservationParams{
		SessionID: sessionID,
		Type:      "progress",
		Title:     fmt.Sprintf("Progress: %s", project),
		Content:   content,
		Project:   project,
		Scope:     "project",
		TopicKey:  topicKey,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to save progress: %v", err)), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Progress updated for %q (ID: %d)", project, id)), nil
}
