package tools

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/HendryAvila/sdd-hoffy/internal/changes"
	"github.com/mark3labs/mcp-go/mcp"
)

// ChangeTool handles the sdd_change MCP tool.
// It creates a new change record with the adaptive pipeline —
// selecting the stage flow based on (type, size).
type ChangeTool struct {
	store changes.Store
}

// NewChangeTool creates a ChangeTool with the given change store.
func NewChangeTool(store changes.Store) *ChangeTool {
	return &ChangeTool{store: store}
}

// Definition returns the MCP tool definition for registration.
func (t *ChangeTool) Definition() mcp.Tool {
	return mcp.NewTool("sdd_change",
		mcp.WithDescription(
			"Create a new change in the adaptive SDD pipeline. "+
				"Each change has a type (feature, fix, refactor, enhancement) and "+
				"size (small, medium, large) that determine which pipeline stages are required. "+
				"Only one active change is allowed at a time. "+
				"Does NOT require sdd_init_project — works independently.",
		),
		mcp.WithString("type",
			mcp.Required(),
			mcp.Description("The kind of work: feature (new capability), fix (bug fix), "+
				"refactor (restructure without behavior change), enhancement (improve existing feature)"),
			mcp.Enum("feature", "fix", "refactor", "enhancement"),
		),
		mcp.WithString("size",
			mcp.Required(),
			mcp.Description("Complexity determines the number of pipeline stages: "+
				"small (3 stages — quick changes), medium (4 stages — moderate changes), "+
				"large (5-6 stages — complex changes requiring full spec + design)"),
			mcp.Enum("small", "medium", "large"),
		),
		mcp.WithString("description",
			mcp.Required(),
			mcp.Description("Brief description of the change. Used to generate the change ID (slug). "+
				"Example: 'Fix FTS5 empty query crash' → change ID 'fix-fts5-empty-query-crash'"),
		),
	)
}

// Handle processes the sdd_change tool call.
func (t *ChangeTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	changeType := changes.ChangeType(req.GetString("type", ""))
	changeSize := changes.ChangeSize(req.GetString("size", ""))
	description := req.GetString("description", "")

	// Validate required fields.
	if err := changes.ValidateType(changeType); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if err := changes.ValidateSize(changeSize); err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	if strings.TrimSpace(description) == "" {
		return mcp.NewToolResultError("'description' is required — briefly describe the change"), nil
	}

	projectRoot, err := findProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("finding project root: %w", err)
	}

	// Guard: only one active change at a time.
	active, err := t.store.LoadActive(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("checking active changes: %w", err)
	}
	if active != nil {
		return mcp.NewToolResultError(fmt.Sprintf(
			"An active change already exists: %q (%s/%s, stage: %s). "+
				"Complete or archive it before starting a new one.",
			active.ID, active.Type, active.Size, active.CurrentStage,
		)), nil
	}

	// Look up stage flow.
	flow, err := changes.StageFlow(changeType, changeSize)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid flow: %v", err)), nil
	}

	// Build stage entries — first stage starts in_progress.
	now := time.Now().UTC().Format(time.RFC3339)
	stageEntries := make([]changes.StageEntry, len(flow))
	for i, stage := range flow {
		status := "pending"
		startedAt := ""
		if i == 0 {
			status = "in_progress"
			startedAt = now
		}
		stageEntries[i] = changes.StageEntry{
			Name:      stage,
			Status:    status,
			StartedAt: startedAt,
		}
	}

	// Create change record.
	changeID := changes.Slugify(description)
	change := &changes.ChangeRecord{
		ID:           changeID,
		Type:         changeType,
		Size:         changeSize,
		Description:  description,
		Stages:       stageEntries,
		CurrentStage: flow[0],
		ADRs:         []string{},
		Status:       changes.StatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := t.store.Create(projectRoot, change); err != nil {
		return nil, fmt.Errorf("creating change: %w", err)
	}

	// Build response.
	var stageList strings.Builder
	for i, s := range flow {
		marker := "⬜"
		if i == 0 {
			marker = "🔄"
		}
		fmt.Fprintf(&stageList, "  %s %s", marker, s)
		if i < len(flow)-1 {
			stageList.WriteString(" →\n")
		} else {
			stageList.WriteString("\n")
		}
	}

	response := fmt.Sprintf(
		"# Change Created\n\n"+
			"**ID:** `%s`\n"+
			"**Type:** %s\n"+
			"**Size:** %s\n"+
			"**Description:** %s\n"+
			"**Status:** active\n\n"+
			"## Pipeline (%d stages)\n\n"+
			"%s\n"+
			"## Next Step\n\n"+
			"Current stage: **%s**\n\n"+
			"Generate the content for the `%s` stage, then call `sdd_change_advance` "+
			"with the content to save it and move to the next stage.",
		change.ID, changeType, changeSize, description,
		len(flow), stageList.String(),
		flow[0], flow[0],
	)

	return mcp.NewToolResultText(response), nil
}
