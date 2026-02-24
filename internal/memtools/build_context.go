package memtools

import (
	"context"
	"fmt"
	"strings"

	"github.com/HendryAvila/Hoofy/internal/memory"
	"github.com/mark3labs/mcp-go/mcp"
)

// BuildContextTool handles the mem_build_context MCP tool.
type BuildContextTool struct {
	store *memory.Store
}

// NewBuildContextTool creates a BuildContextTool with the given memory store.
func NewBuildContextTool(store *memory.Store) *BuildContextTool {
	return &BuildContextTool{store: store}
}

// Definition returns the MCP tool definition for mem_build_context.
func (t *BuildContextTool) Definition() mcp.Tool {
	return mcp.NewTool("mem_build_context",
		mcp.WithDescription(
			"Traverse the knowledge graph from a starting observation. "+
				"Follows relations bidirectionally up to the specified depth, "+
				"returning connected observations with their relation types and directions. "+
				"Use this to understand the full context around a decision, bug fix, or pattern.",
		),
		mcp.WithNumber("observation_id",
			mcp.Required(),
			mcp.Description("The observation ID to start traversal from"),
		),
		mcp.WithNumber("depth",
			mcp.Description("How many levels deep to traverse (default: 2, max: 5)"),
		),
	)
}

// Handle processes the mem_build_context tool call.
func (t *BuildContextTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	obsID := intArg(req, "observation_id", 0)
	if obsID == 0 {
		return mcp.NewToolResultError("'observation_id' is required"), nil
	}

	depth := intArg(req, "depth", 2)

	result, err := t.store.BuildContext(int64(obsID), depth)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to build context: %v", err)), nil
	}

	return mcp.NewToolResultText(formatContextResult(result)), nil
}

// formatContextResult renders a ContextResult as readable markdown.
func formatContextResult(r *memory.ContextResult) string {
	var b strings.Builder

	// Root info
	project := ""
	if r.Root.Project != nil {
		project = *r.Root.Project
	}
	fmt.Fprintf(&b, "# Context Graph for #%d: %q\n\n", r.Root.ID, r.Root.Title)
	fmt.Fprintf(&b, "**Type:** %s\n", r.Root.Type)
	if project != "" {
		fmt.Fprintf(&b, "**Project:** %s\n", project)
	}
	fmt.Fprintf(&b, "**Created:** %s\n\n", r.Root.CreatedAt)

	if len(r.Connected) == 0 {
		b.WriteString("No relations found for this observation.\n")
		return b.String()
	}

	// Group by depth
	byDepth := make(map[int][]memory.ContextNode)
	for _, n := range r.Connected {
		byDepth[n.Depth] = append(byDepth[n.Depth], n)
	}

	for d := 1; d <= r.MaxDepth; d++ {
		nodes, ok := byDepth[d]
		if !ok {
			continue
		}

		label := "Direct Relations"
		if d > 1 {
			label = fmt.Sprintf("Depth %d Relations", d)
		}
		fmt.Fprintf(&b, "## %s (depth %d)\n\n", label, d)

		for _, n := range nodes {
			arrow := "→"
			if n.Direction == "incoming" {
				arrow = "←"
			}
			fmt.Fprintf(&b, "- %s #%d [%s] %q (%s)", arrow, n.ID, n.Type, n.Title, n.RelationType)
			if n.Note != "" {
				fmt.Fprintf(&b, " — %s", n.Note)
			}
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	fmt.Fprintf(&b, "**Total:** %d connected observations across %d level(s)\n", r.TotalNodes, r.MaxDepth)

	return b.String()
}
