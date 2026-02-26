package memtools

import (
	"context"

	"github.com/HendryAvila/Hoofy/internal/memory"
	"github.com/mark3labs/mcp-go/mcp"
)

// ContextTool handles the mem_context MCP tool.
type ContextTool struct {
	store *memory.Store
}

// NewContextTool creates a ContextTool.
func NewContextTool(store *memory.Store) *ContextTool {
	return &ContextTool{store: store}
}

// Definition returns the MCP tool definition for mem_context.
func (t *ContextTool) Definition() mcp.Tool {
	return mcp.NewTool("mem_context",
		mcp.WithDescription(
			"Get recent memory context from previous sessions. Shows recent sessions and "+
				"observations to understand what was done before.",
		),
		mcp.WithString("project",
			mcp.Description("Filter by project (omit for all projects)"),
		),
		mcp.WithString("scope",
			mcp.Description("Filter observations by scope: project (default) or personal"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Number of observations to retrieve (default: 20)"),
		),
		mcp.WithString("detail_level",
			mcp.Description(
				"Level of detail: 'summary' (titles and metadata only — minimal tokens), "+
					"'standard' (default — truncated content snippets), "+
					"'full' (complete untruncated content for deep analysis).",
			),
			mcp.Enum(memory.DetailLevelValues()...),
		),
		mcp.WithString("namespace",
			mcp.Description("Optional sub-agent namespace filter (e.g. 'subagent/task-123'). When set, only returns context from this namespace."),
		),
	)
}

// Handle processes the mem_context tool call.
func (t *ContextTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	project := req.GetString("project", "")
	scope := req.GetString("scope", "")
	limit := intArg(req, "limit", 0)
	detailLevel := memory.ParseDetailLevel(req.GetString("detail_level", ""))
	namespace := req.GetString("namespace", "")

	formatted, err := t.store.FormatContextDetailed(project, scope, memory.ContextFormatOptions{
		DetailLevel: detailLevel,
		Limit:       limit,
		Namespace:   namespace,
	})
	if err != nil {
		return mcp.NewToolResultText("No memory context available."), nil
	}

	if formatted == "" {
		return mcp.NewToolResultText("No memory context available yet. Start saving observations with mem_save."), nil
	}

	// Append footer hint for summary mode.
	if detailLevel == memory.DetailSummary {
		formatted += memory.SummaryFooter
	}

	// Navigation hint when observations are capped by limit.
	total, err := t.store.CountObservations(project, scope, namespace)
	if err == nil {
		// Use effective limit: explicit or default (20).
		effectiveLimit := limit
		if effectiveLimit <= 0 {
			effectiveLimit = 20
		}
		showing := effectiveLimit
		if total < showing {
			showing = total
		}
		formatted += memory.NavigationHint(showing, total,
			"Use mem_search to find specific memories.")
	}

	return mcp.NewToolResultText(formatted), nil
}
