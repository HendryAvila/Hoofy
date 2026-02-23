package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/HendryAvila/sdd-hoffy/internal/changes"
	"github.com/mark3labs/mcp-go/mcp"
)

// ChangeStatusTool handles the sdd_change_status MCP tool.
// It shows the current state of a change or the active change.
type ChangeStatusTool struct {
	store changes.Store
}

// NewChangeStatusTool creates a ChangeStatusTool with the given change store.
func NewChangeStatusTool(store changes.Store) *ChangeStatusTool {
	return &ChangeStatusTool{store: store}
}

// Definition returns the MCP tool definition for registration.
func (t *ChangeStatusTool) Definition() mcp.Tool {
	return mcp.NewTool("sdd_change_status",
		mcp.WithDescription(
			"Show the current state of a change. If `change_id` is provided, "+
				"shows that specific change. Otherwise, shows the active change. "+
				"Returns stage progress, artifact sizes, and ADRs captured.",
		),
		mcp.WithString("change_id",
			mcp.Description("Specific change ID to inspect. If omitted, shows the active change."),
		),
	)
}

// Handle processes the sdd_change_status tool call.
func (t *ChangeStatusTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	changeID := req.GetString("change_id", "")

	projectRoot, err := findProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("finding project root: %w", err)
	}

	var change *changes.ChangeRecord
	if changeID != "" {
		change, err = t.store.Load(projectRoot, changeID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Change %q not found: %v", changeID, err)), nil
		}
	} else {
		change, err = t.store.LoadActive(projectRoot)
		if err != nil {
			return nil, fmt.Errorf("loading active change: %w", err)
		}
		if change == nil {
			return mcp.NewToolResultError("No active change found. Create one with `sdd_change` first."), nil
		}
	}

	// Build stage progress table.
	var stageTable strings.Builder
	stageTable.WriteString("| Stage | Status | Artifact |\n")
	stageTable.WriteString("|-------|--------|----------|\n")

	// Determine the change directory base.
	changeDir := changes.ChangePath(projectRoot, change.ID)
	// If archived, look in history.
	if change.Status == changes.StatusArchived {
		changeDir = filepath.Join(changes.HistoryPath(projectRoot), change.ID)
	}

	for _, s := range change.Stages {
		marker := "⬜"
		switch s.Status {
		case "completed":
			marker = "✅"
		case "in_progress":
			marker = "🔄"
		}

		artifact := "—"
		filename := changes.StageFilename(s.Name)
		if filename != "" {
			filePath := filepath.Join(changeDir, filename)
			if info, statErr := os.Stat(filePath); statErr == nil {
				artifact = fmt.Sprintf("`%s` (%d bytes)", filename, info.Size())
			}
		}

		fmt.Fprintf(&stageTable, "| %s %s | %s | %s |\n", marker, s.Name, s.Status, artifact)
	}

	// ADRs section.
	adrSection := ""
	if len(change.ADRs) > 0 {
		var adrList strings.Builder
		adrList.WriteString("## ADRs\n\n")
		for _, adr := range change.ADRs {
			fmt.Fprintf(&adrList, "- %s\n", adr)
		}
		adrSection = adrList.String() + "\n"
	}

	response := fmt.Sprintf(
		"# Change Status\n\n"+
			"**ID:** `%s`\n"+
			"**Type:** %s\n"+
			"**Size:** %s\n"+
			"**Description:** %s\n"+
			"**Status:** %s\n"+
			"**Created:** %s\n"+
			"**Updated:** %s\n\n"+
			"## Stage Progress\n\n"+
			"%s\n"+
			"%s",
		change.ID, change.Type, change.Size, change.Description,
		change.Status, change.CreatedAt, change.UpdatedAt,
		stageTable.String(),
		adrSection,
	)

	return mcp.NewToolResultText(response), nil
}
