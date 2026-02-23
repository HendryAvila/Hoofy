package memtools

import (
	"context"
	"fmt"
	"strings"

	"github.com/HendryAvila/sdd-hoffy/internal/memory"
	"github.com/mark3labs/mcp-go/mcp"
)

// StatsTool handles the mem_stats MCP tool.
type StatsTool struct {
	store *memory.Store
}

// NewStatsTool creates a StatsTool with the given memory store.
func NewStatsTool(store *memory.Store) *StatsTool {
	return &StatsTool{store: store}
}

// Definition returns the MCP tool definition for mem_stats.
func (t *StatsTool) Definition() mcp.Tool {
	return mcp.NewTool("mem_stats",
		mcp.WithDescription(
			"Show memory system statistics — total sessions, observations, and projects tracked.",
		),
	)
}

// Handle processes the mem_stats tool call.
func (t *StatsTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	stats, err := t.store.Stats()
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to get stats: %v", err)), nil
	}

	var sb strings.Builder
	sb.WriteString("## Memory Statistics\n\n")
	sb.WriteString(fmt.Sprintf("- **Sessions**: %d\n", stats.TotalSessions))
	sb.WriteString(fmt.Sprintf("- **Observations**: %d\n", stats.TotalObservations))
	sb.WriteString(fmt.Sprintf("- **User Prompts**: %d\n", stats.TotalPrompts))

	if len(stats.Projects) > 0 {
		sb.WriteString(fmt.Sprintf("- **Projects** (%d): %s\n", len(stats.Projects), strings.Join(stats.Projects, ", ")))
	} else {
		sb.WriteString("- **Projects**: none\n")
	}

	return mcp.NewToolResultText(sb.String()), nil
}
