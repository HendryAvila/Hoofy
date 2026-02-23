package tools

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/HendryAvila/Hoofy/internal/changes"
	"github.com/mark3labs/mcp-go/mcp"
)

// ChangeAdvanceTool handles the sdd_change_advance MCP tool.
// It is the workhorse of the adaptive pipeline — saves stage content
// and advances the state machine.
type ChangeAdvanceTool struct {
	store  changes.Store
	bridge ChangeObserver
}

// NewChangeAdvanceTool creates a ChangeAdvanceTool with the given change store.
func NewChangeAdvanceTool(store changes.Store) *ChangeAdvanceTool {
	return &ChangeAdvanceTool{store: store}
}

// SetBridge injects an optional ChangeObserver for memory persistence.
func (t *ChangeAdvanceTool) SetBridge(obs ChangeObserver) { t.bridge = obs }

// Definition returns the MCP tool definition for registration.
func (t *ChangeAdvanceTool) Definition() mcp.Tool {
	return mcp.NewTool("sdd_change_advance",
		mcp.WithDescription(
			"Save content for the current stage and advance to the next stage. "+
				"This is the workhorse tool for the adaptive change pipeline. "+
				"It saves the AI-generated content as a markdown file, advances the state machine, "+
				"and reports the next stage. When the final stage (verify) is completed, "+
				"the change is marked as completed.",
		),
		mcp.WithString("content",
			mcp.Required(),
			mcp.Description("The AI-generated content for the current stage. "+
				"Must be actual content, not placeholders. Written as <stage>.md in the change directory."),
		),
		mcp.WithString("title",
			mcp.Description("Optional title for the stage content. "+
				"Used in the response and memory observation."),
		),
	)
}

// Handle processes the sdd_change_advance tool call.
func (t *ChangeAdvanceTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	content := req.GetString("content", "")
	title := req.GetString("title", "")

	if strings.TrimSpace(content) == "" {
		return mcp.NewToolResultError("'content' is required — provide the AI-generated content for the current stage"), nil
	}

	projectRoot, err := findProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("finding project root: %w", err)
	}

	active, err := t.store.LoadActive(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("loading active change: %w", err)
	}
	if active == nil {
		return mcp.NewToolResultError("No active change found. Create one with `sdd_change` first."), nil
	}

	// Determine current stage and its filename.
	currentStage := active.CurrentStage
	filename := changes.StageFilename(currentStage)
	if filename == "" {
		return nil, fmt.Errorf("unknown stage %q — no filename mapping", currentStage)
	}

	// Write content to sdd/changes/<id>/<stage>.md
	changeDir := changes.ChangePath(projectRoot, active.ID)
	stagePath := filepath.Join(changeDir, filename)
	if err := writeStageFile(stagePath, content); err != nil {
		return nil, fmt.Errorf("writing %s: %w", filename, err)
	}

	// Check if this is the final stage (verify).
	isLast := changes.IsLastStage(active)

	if isLast {
		// Final stage — complete the change.
		if err := changes.CompleteChange(active); err != nil {
			return nil, fmt.Errorf("completing change: %w", err)
		}
	} else {
		// Advance to the next stage.
		if err := changes.Advance(active); err != nil {
			return nil, fmt.Errorf("advancing change: %w", err)
		}
	}

	// Persist updated change record.
	if err := t.store.Save(projectRoot, active); err != nil {
		return nil, fmt.Errorf("saving change: %w", err)
	}

	// Notify bridge.
	notifyChangeObserver(t.bridge, active.ID, currentStage, content)

	// Build response.
	if isLast {
		response := fmt.Sprintf(
			"# Stage Completed: %s\n\n"+
				"✅ **Change completed!**\n\n"+
				"**ID:** `%s`\n"+
				"**Type:** %s\n"+
				"**Size:** %s\n"+
				"**Status:** completed\n\n"+
				"All stages have been completed. The change artifacts are in `sdd/changes/%s/`.\n\n"+
				"You can archive this change with `sdd_change_status` to review it, "+
				"or start a new change with `sdd_change`.",
			currentStage, active.ID, active.Type, active.Size, active.ID,
		)
		return mcp.NewToolResultText(response), nil
	}

	// Build stage progress.
	nextStage := active.CurrentStage
	var stageProgress strings.Builder
	for _, s := range active.Stages {
		marker := "⬜"
		switch s.Status {
		case "completed":
			marker = "✅"
		case "in_progress":
			marker = "🔄"
		}
		fmt.Fprintf(&stageProgress, "  %s %s\n", marker, s.Name)
	}

	titleLine := ""
	if title != "" {
		titleLine = fmt.Sprintf("**Title:** %s\n", title)
	}

	response := fmt.Sprintf(
		"# Stage Completed: %s\n\n"+
			"%s"+
			"Saved to `sdd/changes/%s/%s`\n\n"+
			"## Progress\n\n"+
			"%s\n"+
			"## Next Step\n\n"+
			"Current stage: **%s**\n\n"+
			"Generate the content for the `%s` stage, then call `sdd_change_advance` "+
			"with the content to save it and move to the next stage.",
		currentStage, titleLine, active.ID, filename,
		stageProgress.String(),
		nextStage, nextStage,
	)

	return mcp.NewToolResultText(response), nil
}
