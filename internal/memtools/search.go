package memtools

import (
	"context"
	"fmt"
	"strings"

	"github.com/HendryAvila/sdd-hoffy/internal/memory"
	"github.com/mark3labs/mcp-go/mcp"
)

// SearchTool handles the mem_search MCP tool.
type SearchTool struct {
	store *memory.Store
}

// NewSearchTool creates a SearchTool.
func NewSearchTool(store *memory.Store) *SearchTool {
	return &SearchTool{store: store}
}

// Definition returns the MCP tool definition for mem_search.
func (t *SearchTool) Definition() mcp.Tool {
	return mcp.NewTool("mem_search",
		mcp.WithDescription(
			"Search your persistent memory across all sessions. Use this to find past decisions, "+
				"bugs fixed, patterns used, files changed, or any context from previous coding sessions.",
		),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Search query — natural language or keywords"),
		),
		mcp.WithString("type",
			mcp.Description("Filter by type: tool_use, file_change, command, file_read, search, manual, decision, architecture, bugfix, pattern"),
		),
		mcp.WithString("project",
			mcp.Description("Filter by project name"),
		),
		mcp.WithString("scope",
			mcp.Description("Filter by scope: project (default) or personal"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Max results (default: 10, max: 20)"),
		),
	)
}

// Handle processes the mem_search tool call.
func (t *SearchTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	query := req.GetString("query", "")
	if query == "" {
		return mcp.NewToolResultError("'query' is required"), nil
	}

	typ := req.GetString("type", "")
	project := req.GetString("project", "")
	scope := req.GetString("scope", "")
	limit := intArg(req, "limit", 10)

	results, err := t.store.Search(query, memory.SearchOptions{
		Type:    typ,
		Project: project,
		Scope:   scope,
		Limit:   limit,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
	}

	if len(results) == 0 {
		return mcp.NewToolResultText("No memories found matching your query."), nil
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Found %d memories:\n\n", len(results))

	for i, r := range results {
		project := ""
		if r.Project != nil {
			project = *r.Project
		}
		topicInfo := ""
		if r.TopicKey != nil && *r.TopicKey != "" {
			topicInfo = fmt.Sprintf(" | topic: %s", *r.TopicKey)
		}

		snippet := memory.Truncate(r.Content, 300)

		fmt.Fprintf(&b, "[%d] #%d (%s) - %s\n    %s\n    %s%s | scope: %s\n\n",
			i+1, r.ID, r.Type, r.Title,
			snippet,
			project, topicInfo, r.Scope,
		)
	}

	return mcp.NewToolResultText(b.String()), nil
}
