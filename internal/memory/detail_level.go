// detail_level.go provides shared constants and parsing for the detail_level
// parameter used across memory and SDD tools.
//
// Three verbosity levels enable progressive disclosure (Anthropic, 2025):
//   - summary: minimal tokens — IDs, titles, metadata only
//   - standard: default behavior — truncated content snippets
//   - full: complete untruncated content for deep analysis
package memory

import "fmt"

// Detail level constants.
const (
	DetailSummary  = "summary"
	DetailStandard = "standard"
	DetailFull     = "full"
)

// DetailLevelValues returns the enum values for MCP tool definitions.
// Use this to avoid duplicating the list across tool definitions.
func DetailLevelValues() []string {
	return []string{DetailSummary, DetailStandard, DetailFull}
}

// ParseDetailLevel normalizes a detail_level string, defaulting to "standard"
// for empty or unrecognized values.
func ParseDetailLevel(s string) string {
	switch s {
	case DetailSummary, DetailFull:
		return s
	default:
		return DetailStandard
	}
}

// SummaryFooter is appended to summary-mode responses to guide the AI
// toward progressive disclosure — fetch more detail only when needed.
const SummaryFooter = "\n---\n💡 Use detail_level: standard or full for more detail."

// NavigationHint returns a one-line footer when results are capped by a limit.
// Returns an empty string when all results fit (showing >= total) or total is 0.
// The hint parameter provides tool-specific guidance (e.g., "Use mem_get_observation #ID for full content.").
func NavigationHint(showing, total int, hint string) string {
	if total <= 0 || showing >= total {
		return ""
	}
	if hint != "" {
		return fmt.Sprintf("\n📊 Showing %d of %d. %s", showing, total, hint)
	}
	return fmt.Sprintf("\n📊 Showing %d of %d.", showing, total)
}
