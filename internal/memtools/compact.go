package memtools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/HendryAvila/Hoofy/internal/memory"
	"github.com/mark3labs/mcp-go/mcp"
)

// CompactTool handles the mem_compact MCP tool.
// Dual behavior:
//   - Without compact_ids: identifies stale observation candidates (identify mode)
//   - With compact_ids: batch soft-deletes observations + optionally creates summary (execute mode)
type CompactTool struct {
	store *memory.Store
}

// NewCompactTool creates a CompactTool with the given memory store.
func NewCompactTool(store *memory.Store) *CompactTool {
	return &CompactTool{store: store}
}

// Definition returns the MCP tool definition for mem_compact.
func (t *CompactTool) Definition() mcp.Tool {
	return mcp.NewTool("mem_compact",
		mcp.WithDescription(
			"Identify and compact stale memory observations. "+
				"Dual behavior: call WITHOUT compact_ids to list stale candidates, "+
				"WITH compact_ids (JSON array of IDs) to batch soft-delete them. "+
				"Optionally create a summary observation to replace compacted ones. "+
				"Use this to keep memory clean and reduce noise from old sessions.",
		),
		mcp.WithString("project",
			mcp.Description("Filter by project name"),
		),
		mcp.WithString("scope",
			mcp.Description("Filter by scope: project (default) or personal"),
		),
		mcp.WithNumber("older_than_days",
			mcp.Required(),
			mcp.Description("Only consider observations older than this many days (must be > 0)"),
		),
		mcp.WithString("compact_ids",
			mcp.Description(
				"JSON array of observation IDs to compact, e.g. \"[1, 2, 3]\". "+
					"If omitted, returns stale candidates without deleting. "+
					"If provided, batch soft-deletes these observations.",
			),
		),
		mcp.WithString("summary_title",
			mcp.Description("Title for the summary observation created after compaction"),
		),
		mcp.WithString("summary_content",
			mcp.Description("Content for the summary observation. Requires summary_title."),
		),
		mcp.WithString("session_id",
			mcp.Description("Session ID for the summary observation (default: manual-save)"),
		),
	)
}

// Handle processes the mem_compact tool call.
func (t *CompactTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	olderThanDays := intArg(req, "older_than_days", 0)
	if olderThanDays <= 0 {
		return mcp.NewToolResultError("'older_than_days' is required and must be > 0"), nil
	}

	project := req.GetString("project", "")
	scope := req.GetString("scope", "")
	compactIDsRaw := req.GetString("compact_ids", "")

	if compactIDsRaw == "" {
		return t.handleIdentify(project, scope, olderThanDays)
	}
	return t.handleExecute(req, project, scope, compactIDsRaw)
}

// handleIdentify lists stale observation candidates without deleting.
func (t *CompactTool) handleIdentify(project, scope string, olderThanDays int) (*mcp.CallToolResult, error) {
	stale, err := t.store.FindStaleObservations(project, scope, olderThanDays, 200)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to find stale observations: %v", err)), nil
	}

	if len(stale) == 0 {
		return mcp.NewToolResultText(
			fmt.Sprintf("No stale observations found older than %d days.", olderThanDays),
		), nil
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "# Stale Observations (older than %d days)\n\n", olderThanDays)
	fmt.Fprintf(&sb, "Found **%d** candidates for compaction:\n\n", len(stale))

	// Collect IDs for the compact_ids hint
	ids := make([]int64, len(stale))
	for i, obs := range stale {
		ids[i] = obs.ID
		proj := ""
		if obs.Project != nil {
			proj = *obs.Project
		}
		fmt.Fprintf(&sb, "- **ID %d** | %s | %s | `%s` | created: %s\n",
			obs.ID,
			memory.Truncate(obs.Title, 50),
			obs.Type,
			proj,
			obs.CreatedAt,
		)
	}

	// Provide a ready-to-use compact_ids value
	idsJSON, _ := json.Marshal(ids)
	fmt.Fprintf(&sb, "\n---\nTo compact these, call `mem_compact` again with `compact_ids: %q`", string(idsJSON))

	totalCount, _ := t.store.CountObservations(project, scope)
	fmt.Fprintf(&sb, "\n%s", memory.NavigationHint(len(stale), totalCount, ""))

	return mcp.NewToolResultText(sb.String()), nil
}

// handleExecute batch soft-deletes observations and optionally creates a summary.
func (t *CompactTool) handleExecute(req mcp.CallToolRequest, project, scope, compactIDsRaw string) (*mcp.CallToolResult, error) {
	// Parse compact_ids JSON array
	var ids []int64
	if err := json.Unmarshal([]byte(compactIDsRaw), &ids); err != nil {
		return mcp.NewToolResultError(
			fmt.Sprintf("'compact_ids' must be a valid JSON array of integers, e.g. \"[1, 2, 3]\". Parse error: %v", err),
		), nil
	}
	if len(ids) == 0 {
		return mcp.NewToolResultError("'compact_ids' array is empty"), nil
	}

	summaryTitle := req.GetString("summary_title", "")
	summaryContent := req.GetString("summary_content", "")

	if summaryContent != "" && summaryTitle == "" {
		return mcp.NewToolResultError("'summary_content' requires 'summary_title'"), nil
	}

	result, err := t.store.CompactObservations(memory.CompactParams{
		IDs:            ids,
		SummaryTitle:   summaryTitle,
		SummaryContent: summaryContent,
		Project:        project,
		Scope:          scope,
		SessionID:      req.GetString("session_id", ""),
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("compaction failed: %v", err)), nil
	}

	var sb strings.Builder
	sb.WriteString("# Compaction Complete\n\n")
	fmt.Fprintf(&sb, "- **Deleted**: %d observations\n", result.DeletedCount)
	fmt.Fprintf(&sb, "- **Before**: %d total observations\n", result.TotalBefore)
	fmt.Fprintf(&sb, "- **After**: %d total observations\n", result.TotalAfter)

	if result.SummaryID != nil {
		fmt.Fprintf(&sb, "- **Summary**: created as observation ID %d (type: compaction_summary)\n", *result.SummaryID)
	}

	if result.DeletedCount < len(ids) {
		skipped := len(ids) - result.DeletedCount
		fmt.Fprintf(&sb, "\n⚠️ %d ID(s) were skipped (already deleted or not found).\n", skipped)
	}

	return mcp.NewToolResultText(sb.String()), nil
}
