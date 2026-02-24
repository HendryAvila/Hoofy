package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/HendryAvila/Hoofy/internal/memory"
	"github.com/mark3labs/mcp-go/mcp"
)

// ExploreTool handles the sdd_explore MCP tool.
// It captures structured pre-pipeline context and saves it as a memory
// observation with type=explore and topic_key upsert support.
type ExploreTool struct {
	store *memory.Store
}

// NewExploreTool creates an ExploreTool with the given memory store.
func NewExploreTool(store *memory.Store) *ExploreTool {
	return &ExploreTool{store: store}
}

// Definition returns the MCP tool definition for registration.
func (t *ExploreTool) Definition() mcp.Tool {
	return mcp.NewTool("sdd_explore",
		mcp.WithDescription(
			"Explore and capture structured context BEFORE starting a pipeline. "+
				"Use this to record goals, constraints, preferences, unknowns, and decisions "+
				"from pre-planning discussions. Saves to memory with topic_key upsert — "+
				"call multiple times as context evolves. "+
				"The AI should call this BEFORE sdd_init_project or sdd_change.",
		),
		mcp.WithString("title",
			mcp.Required(),
			mcp.Description("Short, searchable title for the exploration context"),
		),
		mcp.WithString("goals",
			mcp.Description("What the user wants to achieve"),
		),
		mcp.WithString("constraints",
			mcp.Description("Technical, business, or time limitations"),
		),
		mcp.WithString("preferences",
			mcp.Description("Architecture style, tech stack opinions, patterns preferred"),
		),
		mcp.WithString("unknowns",
			mcp.Description("Things still unclear or undecided"),
		),
		mcp.WithString("decisions",
			mcp.Description("Choices already made during exploration"),
		),
		mcp.WithString("context",
			mcp.Description("Free-form additional context that doesn't fit other categories"),
		),
		mcp.WithString("project",
			mcp.Description("Project name for filtering"),
		),
		mcp.WithString("scope",
			mcp.Description("Scope for this observation: project (default) or personal"),
		),
		mcp.WithString("session_id",
			mcp.Description("Session ID to associate with (default: manual-save)"),
		),
	)
}

// exploreCategories defines the ordered list of content sections.
var exploreCategories = []string{
	"Goals", "Constraints", "Preferences", "Unknowns", "Decisions", "Context",
}

// Handle processes the sdd_explore tool call.
func (t *ExploreTool) Handle(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	title := strings.TrimSpace(req.GetString("title", ""))
	if title == "" {
		return mcp.NewToolResultError("'title' is required"), nil
	}

	// Collect content sections from parameters.
	incoming := map[string]string{
		"Goals":       strings.TrimSpace(req.GetString("goals", "")),
		"Constraints": strings.TrimSpace(req.GetString("constraints", "")),
		"Preferences": strings.TrimSpace(req.GetString("preferences", "")),
		"Unknowns":    strings.TrimSpace(req.GetString("unknowns", "")),
		"Decisions":   strings.TrimSpace(req.GetString("decisions", "")),
		"Context":     strings.TrimSpace(req.GetString("context", "")),
	}

	if !hasAnyContent(incoming) {
		return mcp.NewToolResultError(
			"At least one context field (goals, constraints, preferences, unknowns, decisions, context) is required",
		), nil
	}

	project := req.GetString("project", "")
	scope := req.GetString("scope", "project")
	sessionID := req.GetString("session_id", "manual-save")

	topicKey := memory.SuggestTopicKey("explore", title, "")

	// Check for existing observation to merge with.
	existing, err := t.store.FindByTopicKey(topicKey, project, scope)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to check existing context: %v", err)), nil
	}

	action := "Created"
	merged := incoming
	if existing != nil {
		action = fmt.Sprintf("Updated (revision #%d)", existing.RevisionCount+1)
		parsed := parseExploreContent(existing.Content)
		merged = mergeExploreSections(parsed, incoming)
	}

	content := formatExploreContent(merged)

	id, err := t.store.AddObservation(memory.AddObservationParams{
		SessionID: sessionID,
		Type:      "explore",
		Title:     title,
		Content:   content,
		Project:   project,
		Scope:     scope,
		TopicKey:  topicKey,
	})
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("failed to save exploration context: %v", err)), nil
	}

	// Build response.
	var sb strings.Builder
	sb.WriteString("## Exploration Context Saved\n\n")
	sb.WriteString(fmt.Sprintf("**Title:** %s\n", title))
	sb.WriteString(fmt.Sprintf("**Topic Key:** %s\n", topicKey))
	sb.WriteString(fmt.Sprintf("**Action:** %s\n", action))
	sb.WriteString(fmt.Sprintf("**ID:** %d\n\n", id))

	sb.WriteString("### Captured Context\n\n")
	sb.WriteString(content)

	sb.WriteString("\n### Suggested Next Steps\n\n")
	sb.WriteString("- When ready to start a new project: use `sdd_init_project`\n")
	sb.WriteString("- When ready to modify existing code: use `sdd_change`\n")
	sb.WriteString("- To add more context: call `sdd_explore` again with the same title\n")

	// Type/size suggestion.
	allText := gatherAllText(merged)
	sugType, sugSize, reasoning := suggestChangeType(allText)
	sb.WriteString("\n### Type/Size Suggestion\n\n")
	sb.WriteString(fmt.Sprintf("- **Suggested type:** %s — %s\n", sugType, reasoning))
	sb.WriteString(fmt.Sprintf("- **Suggested size:** %s\n", sugSize))

	return mcp.NewToolResultText(sb.String()), nil
}

// ─── Private Helpers ────────────────────────────────────────────────────────

// hasAnyContent returns true if at least one section has non-empty content.
func hasAnyContent(sections map[string]string) bool {
	for _, v := range sections {
		if v != "" {
			return true
		}
	}
	return false
}

// formatExploreContent formats sections into ordered markdown.
func formatExploreContent(sections map[string]string) string {
	var sb strings.Builder
	for _, cat := range exploreCategories {
		v := sections[cat]
		if v == "" {
			continue
		}
		sb.WriteString(fmt.Sprintf("## %s\n\n%s\n\n", cat, v))
	}
	return strings.TrimRight(sb.String(), "\n")
}

// parseExploreContent parses structured markdown back into a section map.
func parseExploreContent(md string) map[string]string {
	result := make(map[string]string)
	if md == "" {
		return result
	}

	lines := strings.Split(md, "\n")
	var currentSection string
	var currentContent []string

	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			// Flush previous section.
			if currentSection != "" {
				result[currentSection] = strings.TrimSpace(strings.Join(currentContent, "\n"))
			}
			currentSection = strings.TrimSpace(strings.TrimPrefix(line, "## "))
			currentContent = nil
			continue
		}
		if currentSection != "" {
			currentContent = append(currentContent, line)
		}
	}
	// Flush last section.
	if currentSection != "" {
		result[currentSection] = strings.TrimSpace(strings.Join(currentContent, "\n"))
	}
	return result
}

// mergeExploreSections merges new sections over existing ones.
// Non-empty new values override; empty new values preserve existing.
func mergeExploreSections(existing, incoming map[string]string) map[string]string {
	result := make(map[string]string, len(exploreCategories))
	for _, cat := range exploreCategories {
		if v := incoming[cat]; v != "" {
			result[cat] = v
		} else if v := existing[cat]; v != "" {
			result[cat] = v
		}
	}
	return result
}

// gatherAllText concatenates all non-empty section values into a single string.
func gatherAllText(sections map[string]string) string {
	var parts []string
	for _, cat := range exploreCategories {
		if v := sections[cat]; v != "" {
			parts = append(parts, v)
		}
	}
	return strings.Join(parts, " ")
}

// suggestChangeType applies keyword heuristics to suggest a change type and size.
func suggestChangeType(text string) (suggestedType, suggestedSize, reasoning string) {
	lower := strings.ToLower(text)

	// Type heuristics — first match wins.
	switch {
	case containsAny(lower, "fix", "bug", "crash", "error", "broken"):
		suggestedType = "fix"
		reasoning = "detected fix/bug-related keywords"
	case containsAny(lower, "refactor", "restructure", "reorganize", "clean up", "cleanup"):
		suggestedType = "refactor"
		reasoning = "detected refactoring-related keywords"
	case containsAny(lower, "improve", "enhance", "optimize", "better", "upgrade"):
		suggestedType = "enhancement"
		reasoning = "detected improvement-related keywords"
	case containsAny(lower, "new", "add", "create", "build", "implement"):
		suggestedType = "feature"
		reasoning = "detected feature-related keywords"
	default:
		suggestedType = "feature"
		reasoning = "no strong signal — defaulting to feature"
	}

	// Size heuristics.
	switch {
	case containsAny(lower, "quick", "small", "simple", "trivial", "one-liner", "minor"):
		suggestedSize = "small"
	case containsAny(lower, "complex", "large", "major", "big", "rewrite", "overhaul"):
		suggestedSize = "large"
	default:
		suggestedSize = "medium"
	}

	return
}

// containsAny returns true if text contains any of the given substrings.
func containsAny(text string, subs ...string) bool {
	for _, s := range subs {
		if strings.Contains(text, s) {
			return true
		}
	}
	return false
}
