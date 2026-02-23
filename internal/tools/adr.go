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

// ADRTool handles the sdd_adr MCP tool.
// It captures Architecture Decision Records during changes.
type ADRTool struct {
	store  changes.Store
	bridge ChangeObserver
}

// NewADRTool creates an ADRTool with the given change store.
func NewADRTool(store changes.Store) *ADRTool {
	return &ADRTool{store: store}
}

// SetBridge injects an optional ChangeObserver for memory persistence.
func (t *ADRTool) SetBridge(obs ChangeObserver) { t.bridge = obs }

// validADRStatuses contains the allowed ADR status values.
var validADRStatuses = map[string]bool{
	"proposed":   true,
	"accepted":   true,
	"deprecated": true,
	"superseded": true,
}

// Definition returns the MCP tool definition for registration.
func (t *ADRTool) Definition() mcp.Tool {
	return mcp.NewTool("sdd_adr",
		mcp.WithDescription(
			"Capture an Architecture Decision Record (ADR). "+
				"Works with or without an active change. "+
				"With active change: saves ADR file in the change directory and to memory. "+
				"Without active change: saves to memory only. "+
				"ADRs document important architectural decisions with context, "+
				"rationale, and alternatives considered.",
		),
		mcp.WithString("title",
			mcp.Required(),
			mcp.Description("Short title for the decision. Example: 'Use PostgreSQL over MongoDB'"),
		),
		mcp.WithString("context",
			mcp.Required(),
			mcp.Description("Problem context — what situation requires a decision?"),
		),
		mcp.WithString("decision",
			mcp.Required(),
			mcp.Description("What was decided — the actual architectural decision made."),
		),
		mcp.WithString("rationale",
			mcp.Required(),
			mcp.Description("Why this decision was made — the reasoning behind it."),
		),
		mcp.WithString("alternatives_rejected",
			mcp.Description("What other options were considered and why they were rejected."),
		),
		mcp.WithString("status",
			mcp.Description("ADR status: proposed, accepted (default), deprecated, superseded."),
		),
	)
}

// Handle processes the sdd_adr tool call.
func (t *ADRTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	title := req.GetString("title", "")
	adrContext := req.GetString("context", "")
	decision := req.GetString("decision", "")
	rationale := req.GetString("rationale", "")
	alternatives := req.GetString("alternatives_rejected", "")
	status := req.GetString("status", "accepted")

	// Validate required fields.
	if strings.TrimSpace(title) == "" {
		return mcp.NewToolResultError("'title' is required — provide a short title for the decision"), nil
	}
	if strings.TrimSpace(adrContext) == "" {
		return mcp.NewToolResultError("'context' is required — describe the problem context"), nil
	}
	if strings.TrimSpace(decision) == "" {
		return mcp.NewToolResultError("'decision' is required — state what was decided"), nil
	}
	if strings.TrimSpace(rationale) == "" {
		return mcp.NewToolResultError("'rationale' is required — explain why this decision was made"), nil
	}

	// Validate status.
	if !validADRStatuses[status] {
		return mcp.NewToolResultError(fmt.Sprintf(
			"invalid ADR status %q: must be one of: proposed, accepted, deprecated, superseded", status,
		)), nil
	}

	projectRoot, err := findProjectRoot()
	if err != nil {
		return nil, fmt.Errorf("finding project root: %w", err)
	}

	// Build ADR markdown content.
	var content strings.Builder
	fmt.Fprintf(&content, "# %s\n\n", title)
	fmt.Fprintf(&content, "**Status:** %s\n\n", status)
	content.WriteString("## Context\n\n")
	content.WriteString(adrContext + "\n\n")
	content.WriteString("## Decision\n\n")
	content.WriteString(decision + "\n\n")
	content.WriteString("## Rationale\n\n")
	content.WriteString(rationale + "\n\n")
	if alternatives != "" {
		content.WriteString("## Alternatives Rejected\n\n")
		content.WriteString(alternatives + "\n")
	}

	adrContent := content.String()

	// Check for active change.
	active, err := t.store.LoadActive(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("loading active change: %w", err)
	}

	if active != nil {
		// Save ADR to change directory.
		adrNum := len(active.ADRs) + 1
		adrID := fmt.Sprintf("ADR-%03d", adrNum)

		adrsDir := filepath.Join(changes.ChangePath(projectRoot, active.ID), changes.ADRsDir)
		if err := os.MkdirAll(adrsDir, 0o755); err != nil {
			return nil, fmt.Errorf("creating adrs directory: %w", err)
		}

		adrPath := filepath.Join(adrsDir, adrID+".md")
		if err := writeStageFile(adrPath, adrContent); err != nil {
			return nil, fmt.Errorf("writing ADR: %w", err)
		}

		// Update change record.
		active.ADRs = append(active.ADRs, adrID)
		if err := t.store.Save(projectRoot, active); err != nil {
			return nil, fmt.Errorf("saving change: %w", err)
		}

		// Notify bridge with ADR content.
		notifyChangeObserver(t.bridge, active.ID, "adr", adrContent)

		response := fmt.Sprintf(
			"# ADR Captured\n\n"+
				"**ID:** %s\n"+
				"**Title:** %s\n"+
				"**Status:** %s\n"+
				"**Change:** `%s`\n\n"+
				"Saved to `sdd/changes/%s/adrs/%s.md`\n\n"+
				"## Content\n\n%s",
			adrID, title, status, active.ID,
			active.ID, adrID,
			adrContent,
		)
		return mcp.NewToolResultText(response), nil
	}

	// No active change — notify bridge only (memory persistence).
	notifyChangeObserver(t.bridge, "", "adr", adrContent)

	response := fmt.Sprintf(
		"# ADR Captured (standalone)\n\n"+
			"**Title:** %s\n"+
			"**Status:** %s\n\n"+
			"No active change — ADR saved to memory only.\n\n"+
			"## Content\n\n%s",
		title, status, adrContent,
	)
	return mcp.NewToolResultText(response), nil
}
